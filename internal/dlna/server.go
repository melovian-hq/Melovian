// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package dlna

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"melovian/internal/brand"
)

const (
	ssdpAddr      = "239.255.255.250:1900"
	contentDevice = "urn:schemas-upnp-org:device:MediaServer:1"
)

var (
	serverName = brand.Name + " MediaServer"
	dlnaUDN    = "uuid:" + brand.Slug + "-dlna"
)

type CatalogEntry struct {
	ID    string
	Title string
	Kind  string
}

type CatalogProvider interface {
	Entries() []CatalogEntry
}

type Server struct {
	enabled  bool
	host     string
	httpPort int
	baseURL  string
	catalog  CatalogProvider
	httpSrv  *http.Server
	udpConn  *net.UDPConn
	cancel   context.CancelFunc
}

func New(enabled bool, host string, httpPort int, baseURL string, catalog CatalogProvider) *Server {
	if host == "" {
		host = "0.0.0.0"
	}
	if httpPort <= 0 {
		httpPort = 8200
	}
	return &Server{
		enabled:  enabled && catalog != nil,
		host:     host,
		httpPort: httpPort,
		baseURL:  strings.TrimRight(baseURL, "/"),
		catalog:  catalog,
	}
}

func (s *Server) Enabled() bool {
	return s.enabled
}

func (s *Server) Start(ctx context.Context) error {
	if !s.enabled {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	mux := http.NewServeMux()
	mux.HandleFunc("/dlna/description.xml", s.handleDescription)
	mux.HandleFunc("/dlna/content", s.handleContent)

	s.httpSrv = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.host, s.httpPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("dlna http server stopped", "err", err)
		}
	}()
	go s.ssdpLoop(ctx)

	slog.Info("dlna server started", "addr", s.httpSrv.Addr)
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.udpConn != nil {
		_ = s.udpConn.Close()
	}
	if s.httpSrv != nil {
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleDescription(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<?xml version="1.0"?>
<root xmlns="urn:schemas-upnp-org:device-1-0">
  <specVersion><major>1</major><minor>0</minor></specVersion>
  <device>
    <deviceType>%s</deviceType>
    <friendlyName>%s</friendlyName>
    <manufacturer>Quad4 Software</manufacturer>
    <modelName>%s</modelName>
    <UDN>%s</UDN>
    <serviceList>
      <service>
        <serviceType>urn:schemas-upnp-org:service:ContentDirectory:1</serviceType>
        <controlURL>%s/dlna/content</controlURL>
        <eventSubURL>%s/dlna/content</eventSubURL>
        <SCPDURL>%s/dlna/description.xml</SCPDURL>
      </service>
    </serviceList>
  </device>
</root>`, contentDevice, serverName, brand.Name, dlnaUDN, s.baseURL, s.baseURL, s.baseURL)
}

func (s *Server) handleContent(w http.ResponseWriter, _ *http.Request) {
	entries := s.catalog.Entries()
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><DIDL-Lite xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/">`)
	for _, entry := range entries {
		b.WriteString(`<container id="`)
		b.WriteString(xmlEscape(entry.ID))
		b.WriteString(`" restricted="1" searchable="1"><dc:title>`)
		b.WriteString(xmlEscape(entry.Title))
		b.WriteString(`</dc:title><upnp:class>object.container`)
		if entry.Kind == "album" {
			b.WriteString(".album.musicAlbum")
		} else {
			b.WriteString(".person.musicArtist")
		}
		b.WriteString(`</upnp:class></container>`)
	}
	b.WriteString(`</DIDL-Lite>`)
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}

func (s *Server) ssdpLoop(ctx context.Context) {
	addr, err := net.ResolveUDPAddr("udp4", ssdpAddr)
	if err != nil {
		return
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 1900})
	if err != nil {
		slog.Warn("dlna ssdp disabled", "err", err)
		return
	}
	s.udpConn = conn
	location := fmt.Sprintf("%s/dlna/description.xml", s.baseURL)
	message := strings.Join([]string{
		"NOTIFY * HTTP/1.1",
		"HOST: " + ssdpAddr,
		"NT: " + contentDevice,
		"NTS: ssdp:alive",
		"SERVER: " + brand.Name + "/UPnP 1.0",
		"LOCATION: " + location,
		"USN: " + dlnaUDN + "::" + contentDevice,
		"", "",
	}, "\r\n")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = conn.WriteToUDP([]byte(message), addr)
		}
	}
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
		`'`, "&apos;",
	)
	return replacer.Replace(value)
}
