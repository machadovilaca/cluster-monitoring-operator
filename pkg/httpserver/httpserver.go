package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"k8s.io/klog/v2"

	alertmanagement "github.com/openshift/cluster-monitoring-operator/pkg/alert/management"
)

type Server struct {
	server *http.Server
}

func New(addr string, alertsManagementController alertmanagement.Controller) *Server {
	mux := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}

	mux.HandleFunc("GET /health", s.healthHandler)

	mux.Handle("/api/v1/alerting/", http.StripPrefix("/api/v1/alerting", AlertManagementMux(alertsManagementController)))

	return s
}

func (s *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()

	klog.Infof("starting alert management server on %s", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := json.Marshal(map[string]string{"status": "healthy"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}
