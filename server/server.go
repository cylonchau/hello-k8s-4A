package server

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"
)

var WebhookConfig = &Webhook{}

const (
	DefaultPort       = 81
	DefaultLDAPURL    = "ldap://10.0.0.10:389"
	DefaultLDAPBaseDN = "cn=admin,dc=test,dc=com"
	DefaultLDAPBindDN = "ou=searchUser,cn=admin,dc=test,dc=com"
	DefaultLDAPAuth   = "111"
)

type Webhook struct {
	Port       int
	LDAPURL    string
	LDAPBaseDN string
	LDAPBindDN string
	LDAPAuth   string
}

func BuildInitFlags() {
	flagset := flag.CommandLine
	flagset.IntVar(&WebhookConfig.Port, "port", DefaultPort, "serve port")
	flagset.StringVar(&WebhookConfig.LDAPURL, "ldap_url", DefaultLDAPURL, "ldap url")
	flagset.StringVar(&WebhookConfig.LDAPBaseDN, "ldap_base_dn", DefaultLDAPBaseDN, "ldap base dn")
	flagset.StringVar(&WebhookConfig.LDAPBindDN, "ldap_bind_dn", DefaultLDAPBindDN, "ldap bind dn")
	flagset.StringVar(&WebhookConfig.LDAPAuth, "ldap_auth", DefaultLDAPAuth, "password of ldap bind dn")

	klog.InitFlags(flagset)
	flag.Parse()
}

func Run() {
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", WebhookConfig.Port),
	}

	httpserver := http.NewServeMux()
	httpserver.HandleFunc("/validate", serveAdmission)
	httpserver.HandleFunc("/mutate", serveAdmission)
	httpserver.HandleFunc("/authentication", serveAuthentication)
	httpserver.HandleFunc("/authorization", serveAuthorization)
	httpserver.HandleFunc("/auditing", serveAudit)

	httpserver.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		klog.Info(fmt.Sprintf("%s %s", r.RequestURI, "pong"))
		fmt.Fprint(w, "pong")
	})
	server.Handler = httpserver

	go func() {
		if err := server.ListenAndServe(); err != nil {
			klog.Errorf("Failed to listen and serve hello-4A webhook server: %v", err)
		}
	}()

	klog.V(4).Info(fmt.Sprintf("Listening on port %d waiting for requests...", WebhookConfig.Port))
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	klog.Infof("Got shutdown signal, shutting...")
	if err := server.Shutdown(context.Background()); err != nil {
		klog.Errorf("HTTP server Shutdown: %v", err)
	}
}
