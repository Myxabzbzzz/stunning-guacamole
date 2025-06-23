package apiserver

import (
	"dodobackend/internal/app/store"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// APIServer
type APIServer struct {
	config *Config
	logger *logrus.Logger
	router *mux.Router
	Store  *store.Store
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
	if err := apiServer.configureStore(); err != nil {
		return err
	}

	apiServer.logger.Info("Starting API Server")
	return http.ListenAndServe(apiServer.config.BindAddr, apiServer.router)
}

// logger
func (apiServer *APIServer) configureLogger() error {
	level, err := logrus.ParseLevel(apiServer.config.LogLevel)
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

func (apiserver *APIServer) configureStore() error {
	st := store.New(apiserver.config.Store)
	if err := st.Open(); err != nil {
		return err
	}
	apiserver.Store = st
	return nil
}

func (apiServer *APIServer) handleStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}
