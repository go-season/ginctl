package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/go-season/ginctl/pkg/ginctl/doc"
	"github.com/go-season/ginctl/pkg/util"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/spf13/cobra"
	"github.com/swaggo/swag"
	"github.com/swaggo/swag/gen"
)

const yapiRoute = "/api/open/import_data"
const (
	defaultConsulServer         = "http://test-consul-cluster.piggy.xiaozhu.com:8500"
	defaultYapiTokenPathPattern = "/v1/kv/yapi/%s/token"
	defaultYapiServer           = "http://yapi.piggy.xiaozhu.com"
	defaultYapiMerge            = "mergin"
)

var (
	MergePaths       = make(map[string]map[string]map[string]interface{})
	MergeDefinitions = make(map[string]interface{})
	mutex            sync.Mutex
)

type docCmd struct {
	log log.Logger

	verbose          bool
	consulServer     string
	importDocToYapi  bool
	swagDocFile      string
	yapiConfigPath   string
	searchDir        string
	exclude          string
	generalInfo      string
	propertyStrategy string
	output           string
	parseVendor      bool
	parseDependency  bool
	markdownFiles    string
	codeExampleFiles string
	parseInternal    bool
	generatedTime    bool
	parseDepth       int

	mergeCfgFile string
}

func NewDocCmd(f factory.Factory) *cobra.Command {
	cmd := &docCmd{
		log: f.GetLog(),
	}

	docCmd := &cobra.Command{
		Use:   "doc",
		Short: "生成路由API文档",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	docCmd.Flags().BoolVarP(&cmd.importDocToYapi, "importYapi", "i", false, "Is execute import swagger doc to yapi")
	docCmd.Flags().StringVar(&cmd.swagDocFile, "swagDoc", "./docs/swagger.json", "Swagger json doc path, default ./docs/swagger.json")
	docCmd.Flags().StringVar(&cmd.consulServer, "consulServer", "", "Consul address for project pull yapi config")
	docCmd.Flags().StringVar(&cmd.yapiConfigPath, "yc", "./yapi.json", "Yapi server address if there no consul server that need pass, default ./yapi.json")
	docCmd.Flags().StringVarP(&cmd.generalInfo, "generalInfo", "g", "doc.go", "Go file path in which 'swagger general API Info' is written")
	docCmd.Flags().StringVarP(&cmd.searchDir, "dir", "d", "./api/doc", "Directory you want to parse")
	docCmd.Flags().StringVar(&cmd.exclude, "exclude", "", "Exclude directories and files when searching, comma separated")
	docCmd.Flags().StringVarP(&cmd.propertyStrategy, "propertyStrategy", "p", "camelcase", "Property Naming Strategy like snakecase,camelcase,pascalcase")
	docCmd.Flags().StringVarP(&cmd.output, "output", "o", "./docs", "Output directory for all the generated files(swagger.json, swagger.yaml and doc.go)")
	docCmd.Flags().BoolVar(&cmd.parseVendor, "parseVendor", false, "Parse go files in 'vendor' folder, disabled by default")
	docCmd.Flags().BoolVar(&cmd.parseDependency, "parseDependency", true, "Parse go files in outside dependency folder, disabled by default")
	docCmd.Flags().StringVarP(&cmd.markdownFiles, "markdownFiles", "m", "", "Parse folder containing markdown files to use as description, disabled by default")
	docCmd.Flags().StringVarP(&cmd.codeExampleFiles, "codeExampleFiles", "f", "", "Parse folder containing code example files to use for the x-codeSamples extension, disabled by default")
	docCmd.Flags().BoolVar(&cmd.parseInternal, "parseInternal", false, "Parse go files in internal packages, disabled by default")
	docCmd.Flags().BoolVar(&cmd.generatedTime, "generatedTime", false, "Generate timestamp at the top of docs.go, disabled by default")
	docCmd.Flags().BoolVarP(&cmd.verbose, "verbose", "v", false, "Generate timestamp at the top of docs.go, disabled by default")
	docCmd.Flags().IntVar(&cmd.parseDepth, "parseDepth", 2, "Dependency parse depth")
	docCmd.Flags().StringVar(&cmd.mergeCfgFile, "mc", "", "Specified merge doc config dir")

	return docCmd
}

func (cmd *docCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	normalize := doc.NewNormalize(f.GetLog(), cwd+"/", cmd.verbose)
	if err := normalize.Check(); err != nil {
		return err
	}

	dirMap := make(map[string]bool)
	parts := strings.Split(cmd.exclude, ",")
	for _, part := range parts {
		if strings.HasPrefix(part, "./") {
			part = fmt.Sprintf("%s/%s", strings.TrimSuffix(cwd, "/"), part[2:])
		} else {
			part = fmt.Sprintf("%s/%s", strings.TrimSuffix(cwd, "/"), part)
		}
		dirMap[part] = true
	}

	pkgs := doc.NewPackagesDefinitions(doc.WithExcludes(dirMap), doc.WithWorkdir(cwd))
	parser := doc.NewParser(
		doc.WithPackagesDefinitions(pkgs),
		doc.WithWorkDir(cwd),
		doc.WithDebug(cmd.verbose),
	)

	searchDir := fmt.Sprintf("%s/api/rest", cwd)
	err = parser.ParseAPI(searchDir)
	if err != nil {
		return err
	}

	if err = parser.Packages.RangeFiles(parser.ParseCommentInfo); err != nil {
		return err
	}

	docDir := fmt.Sprintf("%s/api/doc", cwd)
	found, err := file.PathExists(docDir)
	if err != nil {
		return err
	}
	if !found {
		os.Mkdir(docDir, 0755)
	}
	importPath := strings.Join(parser.TypePackagePathCache, "\n")
	tpl := `package doc

import (
	%s
)

func main() {}
`
	fs, err := os.Create(fmt.Sprintf("%s/api/doc/doc.go", cwd))
	if err != nil {
		return err
	}
	defer fs.Close()
	fs.WriteString(fmt.Sprintf(tpl, importPath))

	strategy := cmd.propertyStrategy
	switch strategy {
	case swag.CamelCase, swag.SnakeCase, swag.PascalCase:
	default:
		return fmt.Errorf("not supported %s propertyStrategy", strategy)
	}

	err = gen.New().Build(&gen.Config{
		SearchDir:           cmd.searchDir,
		Excludes:            cmd.exclude,
		MainAPIFile:         cmd.generalInfo,
		PropNamingStrategy:  strategy,
		OutputDir:           cmd.output,
		ParseVendor:         cmd.parseVendor,
		ParseDependency:     cmd.parseDependency,
		MarkdownFilesDir:    cmd.markdownFiles,
		ParseInternal:       cmd.parseInternal,
		GeneratedTime:       cmd.generatedTime,
		CodeExampleFilesDir: cmd.codeExampleFiles,
		ParseDepth:          cmd.parseDepth,
	})

	if err != nil {
		return err
	}

	// clearing generate doc template
	defer func() {
		err := os.RemoveAll(fmt.Sprintf("%s/api/doc", cwd))
		if err != nil {
			cmd.log.Fatalf("remove %s failed", fmt.Sprintf("%s/api/doc", cwd))
		}
	}()

	cmd.log.WriteString("\n")
	cmd.log.Done("Generate API documentation successful.")
	cmd.log.WriteString("\n")

	if cmd.importDocToYapi {
		cmd.log.Info("starting import doc to yapi server...")
		var yo YapiOptions
		var yresp YapiResp
		found, err := file.PathExists(cmd.yapiConfigPath)
		if err != nil {
			return err
		}
		if found {
			err := loadParamsFromConfig(cmd.yapiConfigPath, &yo)
			if err != nil {
				return err
			}
			if yo.Merge == "" {
				yo.Merge = "normal"
			}
		} else {
			err := loadParamsFromConsul(cwd, &yo)
			if err != nil {
				return err
			}
		}
		if err = importToYapi(cmd.swagDocFile, cmd.mergeCfgFile, &yo, &yresp); err != nil {
			return err
		}

		if yresp.ErrCode != 0 {
			cmd.log.Fail(yresp.ErrMsg)
		} else {
			cmd.log.WriteString("\n")
			cmd.log.Done("import doc to yapi successful")
			cmd.log.Info(yresp.ErrMsg)
		}
	}

	return nil
}

type MergeCfg struct {
	URL   string   `yaml:"url"`
	Paths []string `yaml:"paths"`
}

type MergeCfgs struct {
	Merge []MergeCfg `yaml:"merge"`
}

func mergeDoc(file string, swagDoc *swagDocJson) error {
	fs, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fs.Close()

	content, err := ioutil.ReadAll(fs)
	if err != nil {
		return err
	}

	var cfgs MergeCfgs

	err = yaml.Unmarshal(content, &cfgs)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	for _, cfg := range cfgs.Merge {
		wg.Add(1)
		go loadSpecifiedDoc(cfg, &wg)
	}
	wg.Wait()

	for pathName, pathObj := range MergePaths {
		swagDoc.Paths[pathName] = pathObj
	}

	for defineName, defineObj := range MergeDefinitions {
		swagDoc.Definitions[defineName] = defineObj
	}

	return nil
}

func loadSpecifiedDoc(cfg MergeCfg, wg *sync.WaitGroup) {
	response, err := http.Get(cfg.URL)
	if err != nil {
		panic(err)
	}

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	var swaggerJson swagDocJson

	err = json.Unmarshal(content, &swaggerJson)
	if err != nil {
		panic(err)
	}

	pathMap := make(map[string]bool)
	for _, path := range cfg.Paths {
		pathMap[path] = true
	}

	paths := swaggerJson.Paths
	for path, pathObj := range paths {
		if _, ok := pathMap[path]; ok {
			MergePaths[path] = pathObj
			for _, obj := range pathObj {
				if obj["parameters"] != nil {
					parameters := obj["parameters"].([]interface{})
					if parameters != nil {
						for _, parameter := range parameters {
							if parameter.(map[string]interface{})["schema"] != nil {
								schema := parameter.(map[string]interface{})["schema"]
								if schema.(map[string]interface{})["$ref"] != "" {
									ref := schema.(map[string]interface{})["$ref"]
									refStruct := strings.Replace(ref.(string), "#/definitions/", "", 1)
									if refObj, ok := swaggerJson.Definitions[refStruct]; ok {
										MergeDefinitions[refStruct] = refObj
									}
								}
							}
						}
					}
				}
				response := obj["responses"]
				if response != nil {
					okResponse := response.(map[string]interface{})["200"]
					schema := okResponse.(map[string]interface{})["schema"]
					ref := schema.(map[string]interface{})["$ref"]
					typ := schema.(map[string]interface{})["type"]
					if ref == nil && typ == "array" {
						items := schema.(map[string]interface{})["items"]
						ref = items.(map[string]interface{})["$ref"]
					}
					if ref != nil {
						refStruct := strings.Replace(ref.(string), "#/definitions/", "", 1)
						if refObj, ok := swaggerJson.Definitions[refStruct]; ok {
							MergeDefinitions[refStruct] = refObj
							parseAllDefinitions(refObj.(map[string]interface{}), swaggerJson)
						}
					}
				}
			}
		}
	}

	wg.Done()
}

func parseAllDefinitions(definition map[string]interface{}, swaggerJson swagDocJson) {
	if properties, ok := definition["properties"]; ok {
		for _, item := range properties.(map[string]interface{}) {
			for k, obj := range item.(map[string]interface{}) {
				if k == "$ref" {
					refStruct := strings.Replace(obj.(string), "#/definitions/", "", 1)
					if refObj, ok := swaggerJson.Definitions[refStruct]; ok {
						MergeDefinitions[refStruct] = refObj
						parseAllDefinitions(refObj.(map[string]interface{}), swaggerJson)
					}
				} else if sobj, ok := obj.(map[string]interface{}); ok {
					if ref, ok := sobj["$ref"]; ok {
						refStruct := strings.Replace(ref.(string), "#/definitions/", "", 1)
						if refObj, ok := swaggerJson.Definitions[refStruct]; ok {
							MergeDefinitions[refStruct] = refObj
							parseAllDefinitions(refObj.(map[string]interface{}), swaggerJson)
						}
					}
				}
			}
		}
	}
}

type YapiOptions struct {
	Token  string `json:"token"`
	Merge  string `json:"merge" default:"normal"`
	Server string `json:"server"`
}

type YapiResp struct {
	ErrCode int         `json:"errcode"`
	ErrMsg  string      `json:"errmsg"`
	Data    interface{} `json:"data"`
}

func loadParamsFromConfig(file string, data interface{}) error {
	fs, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fs.Close()

	jsonParser := json.NewDecoder(fs)
	if err = jsonParser.Decode(data); err != nil {
		return err
	}

	return nil
}

type YapiToken struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

func loadParamsFromConsul(dir string, yo *YapiOptions) error {
	modName := util.GetModuleName(dir)
	if modName == "" {
		return errors.New("un identify mod name, please check mod is correct")
	}

	yapiTokenPath := fmt.Sprintf(defaultYapiTokenPathPattern, modName)
	resp, err := http.Get(fmt.Sprintf("%s%s", defaultConsulServer, yapiTokenPath))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var data []YapiToken
	err = json.Unmarshal(content, &data)
	if len(data) == 0 {
		return errors.New("not found token config in consul")
	}
	tokenByte, err := base64.StdEncoding.DecodeString(data[0].Value)
	if err != nil {
		return err
	}

	yo.Token = string(tokenByte)
	yo.Merge = defaultYapiMerge
	yo.Server = defaultYapiServer

	return nil
}

func importToYapi(swagFile, mergeCfg string, opt *YapiOptions, response *YapiResp) error {
	content, err := getNormalizeSwagDoc(swagFile, mergeCfg)
	if err != nil {
		return err
	}

	params := struct {
		Type  string `json:"type"`
		Token string `json:"token"`
		Json  string `json:"json"`
		Merge string `json:"merge"`
	}{
		Type:  "swagger",
		Token: opt.Token,
		Json:  string(content),
		Merge: opt.Merge,
	}

	jsonValue, err := json.Marshal(params)
	if err != nil {
		return err
	}

	resp, err := http.Post(fmt.Sprintf("%s%s", opt.Server, yapiRoute), "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("import doc failed, code: %d", resp.StatusCode))
	}

	err = json.Unmarshal(body, response)
	if err != nil {
		return err
	}

	return nil
}

type swagDocJson struct {
	Swagger     string                                       `json:"swagger"`
	Info        map[string]interface{}                       `json:"info"`
	Paths       map[string]map[string]map[string]interface{} `json:"paths"`
	Definitions map[string]interface{}                       `json:"definitions"`
}

func getNormalizeSwagDoc(swagFile, mergeCfg string) ([]byte, error) {
	content, err := ioutil.ReadFile(swagFile)
	if err != nil {
		return nil, err
	}

	var swagDoc swagDocJson
	err = json.Unmarshal(content, &swagDoc)
	if err != nil {
		return nil, err
	}

	if mergeCfg != "" {
		mergeDoc(mergeCfg, &swagDoc)
	}

	for _, path := range swagDoc.Paths {
		for _, define := range path {
			srcResps := define["responses"].(map[string]interface{})
			destResps := srcResps
			okStatus := srcResps["200"].(map[string]interface{})
			newOkStatus := map[string]interface{}{
				"description": "请求成功",
				"schema": map[string]interface{}{
					"properties": map[string]interface{}{
						"status": map[string]interface{}{
							"description": "状态码",
							"type":        "integer",
						},
						"errorMsg": map[string]interface{}{
							"description": "错误描述",
							"type":        "string",
						},
						"content": okStatus["schema"],
						"timestamp": map[string]interface{}{
							"description": "响应时间戳",
							"type":        "string",
						},
					},
				},
			}
			destResps["200"] = newOkStatus
			define["responses"] = destResps
		}
	}

	return json.Marshal(swagDoc)
}
