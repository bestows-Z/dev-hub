package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/project"
)

func TestContainerArgsUseRestrictedRuntime(t *testing.T) {
	n := names(1)
	args := strings.Join(containerArgs(n.frontend, n.internalNetwork, "demo:local"), " ")
	for _, option := range []string{"--pull=never", "--network devhub-preview-1-internal", "--memory=256m", "--cpus=0.5", "--pids-limit=128", "--read-only", "--cap-drop=ALL", "--user=10001:10001"} {
		if !strings.Contains(args, option) {
			t.Fatalf("missing restriction %q: %s", option, args)
		}
	}
	if strings.Contains(args, "docker.sock") || strings.Contains(args, "--privileged") || strings.Contains(args, "--publish") {
		t.Fatalf("unsafe container arguments: %s", args)
	}
	gateway := strings.Join(gatewayArgs(n, 8080, 8080), " ")
	if !strings.Contains(gateway, "--publish 127.0.0.1::8080") || !strings.Contains(gateway, "--network devhub-preview-1-gateway-net") {
		t.Fatalf("gateway is not bound to loopback: %s", gateway)
	}
}

func TestParseMappedPortRequiresLoopback(t *testing.T) {
	port, err := parseMappedPort("127.0.0.1:32769")
	if err != nil || port != 32769 {
		t.Fatalf("port=%d err=%v", port, err)
	}
	if _, err := parseMappedPort("0.0.0.0:32769"); err == nil {
		t.Fatal("accepted a publicly bound port")
	}
}

func TestDockerfileBasesRequireLocalImagesAndStages(t *testing.T) {
	for _, test := range []struct {
		name, file string
		ok         bool
	}{
		{"simple", "FROM node:20-alpine\nCOPY . .\n", true},
		{"multi-stage", "FROM node:20-alpine AS build\nRUN node --version\nFROM node:20-alpine\nCOPY --from=build /app /app\n", true},
		{"remote add", "FROM node:20-alpine\nADD https://example.com/file /app\n", false},
		{"remote copy", "FROM node:20-alpine\nCOPY --from=registry.example/image /x /x\n", false},
		{"remote syntax", "# syntax=docker/dockerfile:1\nFROM node:20-alpine\n", false},
		{"mount", "FROM node:20-alpine\nRUN --mount=type=secret,id=key cat /run/secrets/key\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "Dockerfile")
			if err := os.WriteFile(filename, []byte(test.file), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := dockerfileBases(filename)
			if (err == nil) != test.ok {
				t.Fatalf("err=%v, want ok=%v", err, test.ok)
			}
		})
	}
}

func TestStopDockerReportsDaemonFailure(t *testing.T) {
	err := stopDocker(context.Background(), names(1), func(_ context.Context, _ ...string) (string, error) {
		return "", errors.New("cannot connect to Docker daemon")
	})
	if err == nil || !strings.Contains(err.Error(), "cannot connect") {
		t.Fatalf("expected Docker failure, got %v", err)
	}
}

func TestFailedDeployCleansUpAfterOriginalContextExpires(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cleanupCalls := 0
	command := func(commandCtx context.Context, args ...string) (string, error) {
		if len(args) > 1 && args[0] == "network" && args[1] == "create" {
			if strings.Contains(strings.Join(args, " "), "gateway-net") {
				return "", errors.New("gateway network failed")
			}
			return "", nil
		}
		cleanupCalls++
		if commandCtx.Err() != nil {
			return "", errors.New("cleanup used expired context")
		}
		return "", nil
	}
	err := deploy(ctx, nil, project.Project{ID: 1}, "front:local", "back:local", 8080, 8080, command)
	if err == nil || !strings.Contains(err.Error(), "gateway network failed") || cleanupCalls != 5 {
		t.Fatalf("cleanup calls=%d err=%v", cleanupCalls, err)
	}
}

func TestDockerGatewayIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_DOCKER") != "1" {
		t.Skip("set DEVHUB_TEST_DOCKER=1 to run the Docker network integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	n := names(uint64(time.Now().UnixNano()))
	defer stopDocker(ctx, n, docker)
	if _, err := docker(ctx, "network", "create", "--internal", "--driver", "bridge", n.internalNetwork); err != nil {
		t.Fatal(err)
	}
	if _, err := docker(ctx, "network", "create", "--driver", "bridge", n.gatewayNetwork); err != nil {
		t.Fatal(err)
	}
	for _, service := range []struct{ name, response string }{{n.frontend, "frontend-ok"}, {n.backend, "backend-ok"}} {
		args := append(containerArgs(service.name, n.internalNetwork, "node:20-alpine"), "node", "-e", "require('http').createServer((q,r)=>r.end('"+service.response+"')).listen(8080,'0.0.0.0')")
		if _, err := docker(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := docker(ctx, gatewayArgs(n, 8080, 8080)...); err != nil {
		t.Fatal(err)
	}
	if _, err := docker(ctx, "network", "connect", n.internalNetwork, n.gateway); err != nil {
		t.Fatal(err)
	}
	mapping, err := docker(ctx, "port", n.gateway, "8080/tcp")
	if err != nil {
		t.Fatal(err)
	}
	port, err := parseMappedPort(mapping)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct{ path, expected string }{{"/", "frontend-ok"}, {"/backend/", "backend-ok"}, {"/backend?probe=1", "backend-ok"}} {
		if err := waitHTTP(ctx, port, route.path); err != nil {
			t.Fatal(err)
		}
		response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, route.path))
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil || string(body) != route.expected {
			t.Fatalf("route %s: status=%d body=%q err=%v", route.path, response.StatusCode, body, err)
		}
	}
}
