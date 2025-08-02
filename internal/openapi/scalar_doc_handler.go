package openapi

import (
	"html/template"
	"net/http"

	configservice "github.com/froz42/kerbernetes/internal/services/config"
)

const scalarDocHtml = `
<!doctype html>
<html>
  <head>
    <title>Kerbernetes API Doc</title>
    <meta charset="utf-8" />
    <meta
      name="viewport"
      content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <script
      id="api-reference"
      data-url="{{ .APIPrefix }}/openapi.json"
    ></script>

    <script>
      var configuration = {
        theme: 'saturn'
      }

      document.getElementById('api-reference').dataset.configuration =
        JSON.stringify(configuration)
    </script>

    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>
`

type ScalarDocData struct {
	APIPrefix string
}

func ScalarDocHandler(config configservice.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		tmpl, err := template.New("scalarDoc").Parse(scalarDocHtml)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := ScalarDocData{
			APIPrefix: config.APIPrefix,
		}

		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
