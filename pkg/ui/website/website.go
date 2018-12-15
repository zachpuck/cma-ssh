package website

import (
	"mime"
	"net/http"

	"github.com/samsung-cnct/cma-ssh/pkg/ui/website/api"
	"github.com/samsung-cnct/cma-ssh/pkg/ui/website/homepage"
	"github.com/samsung-cnct/cma-ssh/pkg/ui/website/swaggerjson"
	"github.com/samsung-cnct/cma-ssh/pkg/ui/website/swaggerui"
)

func AddWebsiteHandles(mux *http.ServeMux) {
	serveSwagger(mux)
	serveSwaggerJSON(mux)
	serveApi(mux)
	serveHomepage(mux)
}

func serveSwagger(mux *http.ServeMux) {
	_ = mime.AddExtensionType(".svg", "image/svg+xml")

	// Expose files in third_party/swagger-ui/ on <host>/swagger-ui
	fileServer := http.FileServer(swaggerui.SwaggerUI)
	prefix := "/swagger-ui/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
}

func serveSwaggerJSON(mux *http.ServeMux) {
	_ = mime.AddExtensionType(".json", "application/json")

	fileServer := http.FileServer(swaggerjson.Swagger)
	prefix := "/swagger/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
}

func serveApi(mux *http.ServeMux) {

	fileServer := http.FileServer(api.Api)
	prefix := "/protobuf/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
}

func serveHomepage(mux *http.ServeMux) {
	fileServer := http.FileServer(homepage.Homepage)
	prefix := "/"
	mux.Handle(prefix, http.StripPrefix("/", fileServer))
}
