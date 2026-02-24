package main

import (
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"
)

//go:embed template/*
var consoleTemplate embed.FS
var consoleTmpl = template.Must(template.ParseFS(consoleTemplate, "template/*"))

func ServeConsole(fsys fs.FS) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		consoleTmpl.Execute(w, nil)
	})
	mux.HandleFunc("GET /rebuild-index", func(w http.ResponseWriter, r *http.Request) {
		if err := server.BuildRAGIndex(r.Context(), fsys); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "索引构建成功"})
	})

	mux.HandleFunc("GET /net-interfaces", getNetInterfaces)

	panic(http.ListenAndServe(os.Getenv("EASTLAUGH_CONSOLE_ADDR"), mux))
}

type ifaceInfo struct {
	Name  string   `json:"name"`
	MTU   int      `json:"mtu"`
	MAC   string   `json:"mac"`
	Flags string   `json:"flags"`
	Addrs []string `json:"addrs"`
}

func getNetInterfaces(w http.ResponseWriter, r *http.Request) {
	interfaces, _ := net.Interfaces()
	var infos []ifaceInfo
	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()
		addrStrs := make([]string, len(addrs))
		for i, a := range addrs {
			addrStrs[i] = a.String()
		}
		infos = append(infos, ifaceInfo{
			Name:  iface.Name,
			MTU:   iface.MTU,
			MAC:   iface.HardwareAddr.String(),
			Flags: iface.Flags.String(),
			Addrs: addrStrs,
		})
	}
	data, _ := json.MarshalIndent(infos, "", "  ")

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
