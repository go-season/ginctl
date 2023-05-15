package log

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	surveypkg "gopkg.in/AlecAivazis/survey.v1"
)

type QuestionOptions struct {
	Question               string
	DefaultValue           string
	ValidationRegexPattern string
	ValidationMessage      string
	ValidationFunc         func(value string) error
	Options                []string
	IsPassword             bool
	IsMultiSelect          bool
}

var DefaultValidationRegexPattern = regexp.MustCompile("^.*$")

type Survey interface {
	Question(params *QuestionOptions) (string, error)
}

type survey struct{}

func NewSurvey() Survey {
	return &survey{}
}

func (s *survey) Question(params *QuestionOptions) (string, error) {
	var prompt surveypkg.Prompt
	compileRegex := DefaultValidationRegexPattern
	if params.ValidationRegexPattern != "" {
		compileRegex = regexp.MustCompile(params.ValidationRegexPattern)
	}

	if params.IsMultiSelect {
		prompt = &surveypkg.MultiSelect{
			Message: params.Question + "\n",
			Options: params.Options,
		}
	} else if params.Options != nil {
		prompt = &surveypkg.Select{
			Message:  params.Question + "\n",
			Options:  params.Options,
			Default:  params.DefaultValue,
			PageSize: 10,
		}
	} else if params.IsPassword {
		prompt = &surveypkg.Password{
			Message: params.Question,
		}
	} else {
		prompt = &surveypkg.Input{
			Message: params.Question,
			Default: params.DefaultValue,
		}
	}

	question := []*surveypkg.Question{
		{
			Name:   "question",
			Prompt: prompt,
		},
	}

	if params.Options == nil {
		question[0].Validate = func(val interface{}) error {
			str, ok := val.(string)
			if !ok {
				return errors.New("Input was not a string")
			}

			if compileRegex.MatchString(str) == false {
				if params.ValidationMessage != "" {
					return errors.New(params.ValidationMessage)
				}

				return errors.Errorf("Answer has no match pattern: %s", compileRegex.String())
			}

			if params.ValidationFunc != nil {
				err := params.ValidationFunc(str)
				if err != nil {
					if params.ValidationMessage != "" {
						errors.New(params.ValidationMessage)
					}

					return errors.Errorf("%v", err)
				}
			}

			return nil
		}
	}

	if params.IsMultiSelect {
		answer := make([]string, 0)
		err := surveypkg.Ask(question, &answer)
		if err != nil {
			return "", err
		}
		return strings.Join(answer, ","), nil
	} else {
		answer := struct {
			Question string
		}{}
		err := surveypkg.Ask(question, &answer)
		if err != nil {
			return "", err
		}
		return answer.Question, nil
	}
}
