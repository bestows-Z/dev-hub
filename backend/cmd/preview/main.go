package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	pg "github.com/bestows-Z/dev-hub/backend/internal/platform/postgres"
	"github.com/bestows-Z/dev-hub/backend/internal/project"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var imagePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/:@-]{0,255}$`)

type commandRunner func(context.Context, ...string) (string, error)

func docker(ctx context.Context, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "docker", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker %s: %w: %s", args[0], err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func main() {
	if err := run(os.Args[1:], docker); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, command commandRunner) error {
	if len(args) == 0 || (args[0] != "deploy" && args[0] != "stop") {
		return errors.New("usage: go run ./cmd/preview deploy|stop --slug NAME [--frontend-image IMAGE --backend-image IMAGE --frontend-port 8080 --backend-port 8080]")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	slug := flags.String("slug", "", "project slug")
	frontendImage := flags.String("frontend-image", "", "locally built frontend image")
	backendImage := flags.String("backend-image", "", "locally built backend image")
	frontendPort := flags.Int("frontend-port", 8080, "frontend container port")
	backendPort := flags.Int("backend-port", 8080, "backend container port")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if *slug == "" || len(flags.Args()) > 0 {
		return errors.New("a project --slug is required")
	}
	if args[0] == "deploy" && (!imagePattern.MatchString(*frontendImage) || !imagePattern.MatchString(*backendImage) || !validPort(*frontendPort) || !validPort(*backendPort)) {
		return errors.New("deploy needs two local image names and valid container ports (1024–65535)")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	connection, err := pg.New(pg.Config{DSN: cfg.Postgres.DSN(), MaxOpenConns: 2, MaxIdleConns: 1}, zap.NewNop())
	if err != nil {
		return err
	}
	defer connection.Close()
	var item project.Project
	if err := connection.DB.Where("slug = ?", *slug).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("project slug not found")
		}
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if args[0] == "stop" {
		if err := stop(ctx, connection.DB, item, command); err != nil {
			return err
		}
		fmt.Printf("stopped %s\n", item.Slug)
		return nil
	}
	for _, image := range []string{*frontendImage, *backendImage, "node:20-alpine"} {
		if _, err := command(ctx, "image", "inspect", image); err != nil {
			return fmt.Errorf("image %q is not built locally: %w", image, err)
		}
	}
	if err := stop(ctx, connection.DB, item, command); err != nil {
		return err
	}
	if err := deploy(ctx, connection.DB, item, *frontendImage, *backendImage, *frontendPort, *backendPort, command); err != nil {
		return err
	}
	fmt.Printf("deployed %s; publish the project to expose /api/v1/project-runtimes/%s/\n", item.Slug, item.Slug)
	return nil
}

func validPort(port int) bool { return port >= 1024 && port <= 65535 }

type runtimeNames struct {
	internalNetwork string
	gatewayNetwork  string
	frontend        string
	backend         string
	gateway         string
}

func names(id uint64) runtimeNames {
	base := fmt.Sprintf("devhub-preview-%d", id)
	return runtimeNames{base + "-internal", base + "-gateway-net", base + "-frontend", base + "-backend", base + "-gateway"}
}

func stop(ctx context.Context, db *gorm.DB, item project.Project, command commandRunner) error {
	if err := stopDocker(ctx, names(item.ID), command); err != nil {
		return err
	}
	previewURL := ""
	if item.BundlePrefix != "" {
		previewURL = fmt.Sprintf("/api/v1/project-previews/%s/index.html", item.Slug)
	}
	return db.Model(&item).Updates(map[string]any{
		"runtime_status": "stopped", "runtime_frontend_port": 0, "runtime_backend_port": 0,
		"preview_url": previewURL, "backend_url": "",
	}).Error
}

func stopDocker(ctx context.Context, n runtimeNames, command commandRunner) error {
	var failures []error
	for _, name := range []string{n.gateway, n.frontend, n.backend} {
		if _, err := command(ctx, "rm", "-f", name); err != nil && !strings.Contains(err.Error(), "No such container") {
			failures = append(failures, err)
		}
	}
	for _, name := range []string{n.gatewayNetwork, n.internalNetwork} {
		if _, err := command(ctx, "network", "rm", name); err != nil && !strings.Contains(err.Error(), "not found") {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func deploy(ctx context.Context, db *gorm.DB, item project.Project, frontendImage, backendImage string, frontendPort, backendPort int, command commandRunner) (err error) {
	n := names(item.ID)
	if _, err = command(ctx, "network", "create", "--internal", "--driver", "bridge", n.internalNetwork); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			if cleanupErr := stopDocker(cleanupCtx, n, command); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("cleanup failed: %w", cleanupErr))
			}
		}
	}()
	if _, err = command(ctx, "network", "create", "--driver", "bridge", n.gatewayNetwork); err != nil {
		return err
	}
	if _, err = command(ctx, containerArgs(n.backend, n.internalNetwork, backendImage)...); err != nil {
		return err
	}
	if _, err = command(ctx, containerArgs(n.frontend, n.internalNetwork, frontendImage)...); err != nil {
		return err
	}
	if _, err = command(ctx, gatewayArgs(n, frontendPort, backendPort)...); err != nil {
		return err
	}
	if _, err = command(ctx, "network", "connect", n.internalNetwork, n.gateway); err != nil {
		return err
	}
	mapping, err := command(ctx, "port", n.gateway, "8080/tcp")
	if err != nil {
		return err
	}
	hostPort, err := parseMappedPort(mapping)
	if err != nil {
		return err
	}
	for _, route := range []string{"/", "/backend/"} {
		if err = waitHTTP(ctx, hostPort, route); err != nil {
			return err
		}
	}
	return db.Model(&item).Updates(map[string]any{
		"runtime_status": "running", "runtime_frontend_port": hostPort, "runtime_backend_port": hostPort,
		"preview_url": fmt.Sprintf("/api/v1/project-runtimes/%s/", item.Slug),
		"backend_url": fmt.Sprintf("/api/v1/project-runtimes/%s/backend/", item.Slug),
	}).Error
}

func containerArgs(name, network, image string) []string {
	return []string{
		"run", "-d", "--pull=never", "--restart=unless-stopped", "--name", name,
		"--network", network,
		"--memory=256m", "--cpus=0.5", "--pids-limit=128", "--read-only",
		"--tmpfs=/tmp:rw,noexec,nosuid,size=64m", "--security-opt=no-new-privileges:true",
		"--cap-drop=ALL", "--user=10001:10001", image,
	}
}

func gatewayArgs(n runtimeNames, frontendPort, backendPort int) []string {
	return []string{
		"run", "-d", "--pull=never", "--restart=unless-stopped", "--name", n.gateway,
		"--network", n.gatewayNetwork, "--publish", "127.0.0.1::8080",
		"--memory=128m", "--cpus=0.5", "--pids-limit=128", "--read-only",
		"--tmpfs=/tmp:rw,noexec,nosuid,size=32m", "--security-opt=no-new-privileges:true",
		"--cap-drop=ALL", "--user=10001:10001", "node:20-alpine", "node", "-e",
		gatewayScript(n, frontendPort, backendPort),
	}
}

func gatewayScript(n runtimeNames, frontendPort, backendPort int) string {
	return fmt.Sprintf(`const http=require('http');http.createServer((req,res)=>{const incoming=new URL(req.url,'http://preview.local');const backend=incoming.pathname==='/backend'||incoming.pathname.startsWith('/backend/');const host=backend?%q:%q;const port=backend?%d:%d;const upstreamPath=backend?(incoming.pathname.slice(8)||'/')+incoming.search:req.url;const upstream=http.request({hostname:host,port:port,path:upstreamPath,method:req.method,headers:{...req.headers,host:host}},response=>{res.writeHead(response.statusCode,response.headers);response.pipe(res)});upstream.on('error',()=>{res.statusCode=502;res.end('project service unavailable')});req.pipe(upstream)}).listen(8080,'0.0.0.0')`, n.backend, n.frontend, backendPort, frontendPort)
}

func parseMappedPort(mapping string) (int, error) {
	for _, line := range strings.Split(mapping, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "127.0.0.1:") {
			continue
		}
		port, err := strconv.Atoi(strings.TrimPrefix(line, "127.0.0.1:"))
		if err == nil && validPort(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("Docker did not publish a loopback port: %q", mapping)
}

func waitHTTP(ctx context.Context, port int, route string) error {
	client := http.Client{Timeout: time.Second}
	address := fmt.Sprintf("http://127.0.0.1:%d%s", port, route)
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode < 500 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("runtime on port %d did not become healthy: %w", port, ctx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}
}
