package phpgen

func getClientFactoryTpl() []byte {
	return []byte(`<?php

namespace App\SDK;

use BundleLib\GuzzleBundle\Client;
use Psr\Http\Message\ResponseInterface;
use Symfony\Component\HttpFoundation\Response;

class {{.ClientName}} extends Client
{
	{{range .Services}}private {{.ServiceName}};{{end}}

	public function __construct(array $config = [])
	{
		{{range .Services}}$this->{{.ServiceName}} = new {{.ServiceName | ToCamel}}($this);{{end}}

		parent::__construct($config);
	}

	{{range .Services}}public function {{.ServiceName | ToCamel}}()
	{
		return $this->{{.ServiceName}};
	}
	{{end}}

	public function ExtractBody(ResponseInterface $response)
    {
        if (Response::HTTP_OK == $response->getStatusCode()) {
            $result = $response->getBody()->getContents();
            if ($result) {
                return json_decode($result, true);
            }
        }

        return [];
    }

	public function getContent(array $result)
    {
        if (!empty($result) && isset($result['status'])) {
            if (Response::HTTP_OK != $result['status']) {
                throw new \Exception($result['errorMsg'], $result['status']);
            }
            return $result['content'];
        }

        throw new \Exception('client error', 4000);
    }
}
`)
}

func getServiceTpl() {

}
