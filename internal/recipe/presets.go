package recipe

import (
	"slices"
	"strings"

	"github.com/DasDarki/nwm/internal/docker"
)

type ServicePreset struct {
	Image         string
	Description   string
	ContainerPort string
	DataPath      string
	Command       []string
	Ready         []string
	Env           map[string]string
	ServerEnv     map[string]string
}

var servicePresets = map[string]ServicePreset{
	"postgres": {
		Image: "postgres:17", Description: "PostgreSQL database", ContainerPort: "5432", DataPath: "/var/lib/postgresql/data",
		Ready:     []string{"pg_isready", "-h", "127.0.0.1", "-U", "nanos", "-d", "nanos"},
		Env:       map[string]string{"POSTGRES_USER": "nanos", "POSTGRES_PASSWORD": "{password}", "POSTGRES_DB": "nanos"},
		ServerEnv: map[string]string{"DATABASE_URL": "postgres://nanos:{password}@127.0.0.1:{port}/nanos"},
	},
	"mysql": {
		Image: "mysql:8.4", Description: "MySQL database", ContainerPort: "3306", DataPath: "/var/lib/mysql",
		Ready:     []string{"mysqladmin", "ping", "-h", "127.0.0.1", "--silent"},
		Env:       map[string]string{"MYSQL_ROOT_PASSWORD": "{password}", "MYSQL_DATABASE": "nanos"},
		ServerEnv: map[string]string{"DATABASE_URL": "mysql://root:{password}@127.0.0.1:{port}/nanos"},
	},
	"mariadb": {
		Image: "mariadb:11", Description: "MariaDB database", ContainerPort: "3306", DataPath: "/var/lib/mysql",
		Ready:     []string{"mariadb-admin", "ping", "-h", "127.0.0.1", "--silent"},
		Env:       map[string]string{"MARIADB_ROOT_PASSWORD": "{password}", "MARIADB_DATABASE": "nanos"},
		ServerEnv: map[string]string{"DATABASE_URL": "mysql://root:{password}@127.0.0.1:{port}/nanos"},
	},
	"redis": {
		Image: "redis:7-alpine", Description: "Redis cache", ContainerPort: "6379", DataPath: "/data",
		Ready:     []string{"redis-cli", "ping"},
		ServerEnv: map[string]string{"REDIS_URL": "redis://127.0.0.1:{port}"},
	},
	"valkey": {
		Image: "valkey/valkey:8-alpine", Description: "Valkey cache (Redis compatible)", ContainerPort: "6379", DataPath: "/data",
		Ready:     []string{"valkey-cli", "ping"},
		ServerEnv: map[string]string{"REDIS_URL": "redis://127.0.0.1:{port}"},
	},
	"mongo": {
		Image: "mongo:8", Description: "MongoDB database", ContainerPort: "27017", DataPath: "/data/db",
		Ready:     []string{"mongosh", "--quiet", "--host", "127.0.0.1", "--eval", "db.adminCommand('ping').ok"},
		Env:       map[string]string{"MONGO_INITDB_ROOT_USERNAME": "nanos", "MONGO_INITDB_ROOT_PASSWORD": "{password}"},
		ServerEnv: map[string]string{"MONGODB_URL": "mongodb://nanos:{password}@127.0.0.1:{port}"},
	},
	"rabbitmq": {
		Image: "rabbitmq:4-management", Description: "RabbitMQ message broker", ContainerPort: "5672", DataPath: "/var/lib/rabbitmq",
		Ready:     []string{"rabbitmq-diagnostics", "-q", "ping"},
		ServerEnv: map[string]string{"RABBITMQ_URL": "amqp://guest:guest@127.0.0.1:{port}"},
	},
	"meilisearch": {
		Image: "getmeili/meilisearch:v1", Description: "Meilisearch search engine", ContainerPort: "7700", DataPath: "/meili_data",
		ServerEnv: map[string]string{"MEILISEARCH_URL": "http://127.0.0.1:{port}"},
	},
	"minio": {
		Image: "minio/minio", Description: "S3 compatible object storage", ContainerPort: "9000", DataPath: "/data",
		Command:   []string{"server", "/data"},
		Env:       map[string]string{"MINIO_ROOT_USER": "nanos", "MINIO_ROOT_PASSWORD": "{password}"},
		ServerEnv: map[string]string{"S3_ENDPOINT": "http://127.0.0.1:{port}"},
	},
}

func PresetNames() []string {
	names := make([]string, 0, len(servicePresets))
	for name := range servicePresets {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func Preset(name string) (ServicePreset, bool) {
	preset, found := servicePresets[name]
	return preset, found
}

func imageBase(image string) string {
	base := image
	if index := strings.LastIndex(base, "/"); index >= 0 {
		base = base[index+1:]
	}
	if index := strings.IndexAny(base, ":@"); index >= 0 {
		base = base[:index]
	}
	return base
}

func expandRequirement(image string, name string, hostPort string, extraEnv map[string]string) (docker.Service, map[string]string, *Secret) {
	base := imageBase(image)
	preset, known := servicePresets[base]
	if known && !strings.ContainsAny(image, ":@") && !strings.Contains(image, "/") {
		image = preset.Image
	}
	if name == "" {
		name = strings.ToLower(base)
	}
	service := docker.Service{Name: name, Image: image, Env: map[string]string{}, Ready: preset.Ready, Command: preset.Command}
	serverEnv := map[string]string{}
	var secret *Secret
	secretName := strings.ReplaceAll(name, "-", "_") + "_password"
	password := "${secret." + secretName + "}"
	usesPassword := false
	fill := func(value string) string {
		if strings.Contains(value, "{password}") {
			usesPassword = true
		}
		return strings.ReplaceAll(value, "{password}", password)
	}
	if known {
		for key, value := range preset.Env {
			service.Env[key] = fill(value)
		}
		port := preset.ContainerPort
		if hostPort == "" {
			hostPort = port
		}
		service.Ports = []string{hostPort + ":" + port}
		if hostPort == port {
			service.Ports = []string{port}
		}
		if preset.DataPath != "" {
			service.Volumes = []string{name + "-data:" + preset.DataPath}
		}
		for key, value := range preset.ServerEnv {
			serverEnv[key] = fill(strings.ReplaceAll(value, "{port}", hostPort))
		}
	} else if hostPort != "" {
		service.Ports = []string{hostPort}
	}
	for key, value := range extraEnv {
		service.Env[key] = value
	}
	if len(service.Env) == 0 {
		service.Env = nil
	}
	if usesPassword {
		secret = &Secret{Name: secretName}
	}
	return service, serverEnv, secret
}
