package apiserver

import (
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"net/http"
)

// APIServer
type APIServer struct {
	config *Config
	logger *logrus.Logger
	router *mux.Router
}

// NewAPIServer
func NewAPIServer(config *Config) *APIServer {
	return &APIServer{
		config: config,
		logger: logrus.New(),
		router: mux.NewRouter(),
	}

}

// Run
func (apiServer *APIServer) Run() error {
	if err := apiServer.configureLogger(); err != nil {
		return err
	}
	apiServer.configureRouter()
	apiServer.logger.Info("Starting API Server")
	return http.ListenAndServe(apiServer.config.BindAddr, apiServer.router)
}

// logger
func (apiServer *APIServer) configureLogger() error {
	level, err := logrus.ParseLevel(apiServer.config.logLevel)
	if err != nil {
		return err
	}
	apiServer.logger.SetLevel(level)
	return nil
}

// router
func (apiServer *APIServer) configureRouter() {
	apiServer.router.HandleFunc("/status", apiServer.handleStatus())
}

func (apiServer *APIServer) handleStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}
