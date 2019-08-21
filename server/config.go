package server

import (
	"os"

	"github.com/go-redis/redis"
	"gopkg.in/go-yaml/yaml.v2"
)

type CacheConfig struct {
	ServerAddress string `yaml:"serverAddress"`
}

type ElasticsearchConfig struct {
	Host          string            `yaml:"host"`
	Port          int               `yaml:"port"`
	Index         string            `yaml:"index"`
	GeometryField string            `yaml:"geometryField"`
	SourceFields  map[string]string `yaml:"sourceFields"`
}

type LayerConfig struct {
	Name          string               `yaml:"name"`
	Description   string               `yaml:"description"`
	Minzoom       int                  `yaml:"minzoom"`
	Maxzoom       int                  `yaml:"maxzoom"`
	Elasticsearch *ElasticsearchConfig `yaml:"elasticsearch"`
}

type Config struct {
	Cache  *CacheConfig  `yaml:"cache"`
	Layers []LayerConfig `yaml:"layers"`
}

func LoadConfig(configFile *os.File) (*Config, error) {
	dec := yaml.NewDecoder(configFile)
	dec.SetStrict(true)
	var config Config
	err := dec.Decode(&config)
	if err != nil {
		return nil, err
	}
	Logger.Debugf("Loaded config: %+v", config)
	return &config, nil
}

// ConfigOption is a function that changes a configuration setting of the server.Server
type ConfigOption func(s *Server) error

// ConfigFile loads a YAML configuration file from disk to set up the server
func ConfigFile(configFile *os.File) ConfigOption {
	return func(s *Server) error {
		config, err := LoadConfig(configFile)
		if err != nil {
			return err
		}
		if config.Cache != nil {
			s.CacheClient = redis.NewClient(&redis.Options{
				Addr: config.Cache.ServerAddress,
			})
		}
		var layers []Layer
		for _, layerConfig := range config.Layers {
			layer, err := CreateLayer(layerConfig)
			if err != nil {
				return err
			}
			layers = append(layers, *layer)
		}
		s.Layers = layers
		return nil
	}
}

// Port hanges the port number used for serving tile data
func Port(port uint16) ConfigOption {
	return func(s *Server) error {
		s.Port = port
		return nil
	}
}

// InternalPort changes the port number used for administrative endpoints (e.g. healthcheck)
func InternalPort(internalPort uint16) ConfigOption {
	return func(s *Server) error {
		s.InternalPort = internalPort
		return nil
	}
}

// EnableCORS configures the server for CORS (cross-origin resource sharing)
func EnableCORS(s *Server) error {
	s.EnableCORS = true
	return nil
}

// Simplify shapes enable geometry simplification based on the requested zoom level
func SimplifyShapes(s *Server) error {
	s.Simplify = true
	return nil
}

const NOTHING = `
// CacheControl sets a fixed string to be used for the Cache-Control HTTP header
func CacheControl(cacheControl string) ConfigOption {
	return func(s *Server) error {
		s.CacheControl = cacheControl
		return nil
	}
}

// CacheServer sets an address to be used to connect to a Redis Cache Server
func CacheServer(cacheServer string) ConfigOption {
	return func(s *Server) error {
		if cacheServer != "" {
			s.CacheClient = redis.NewClient(&redis.Options{
				Addr: cacheServer,
			})
		}
		return nil
	}
}

// CacheTTL sets the time-to-live for Redis
func CacheTTL(cacheTTLString string) ConfigOption {
	return func(s *Server) error {
		if cacheTTLString == "" {
			return nil
		}
		cacheTTL, err := time.ParseDuration(cacheTTLString)
		if err != nil {
			return err
		}
		s.CacheTTL = cacheTTL
		return nil
	}
}

// ESHost sets the Elasticsearch backend host:port
func ESHost(esHost string) ConfigOption {
	return func(s *Server) error {
		client, err := elastic.NewClient(
			elastic.SetURL(fmt.Sprintf("http://%s", esHost)),
			elastic.SetGzip(true),
			// TODO: Should this be configurable?
			elastic.SetHealthcheckTimeoutStartup(30*time.Second),
		)
		s.ES = client
		return err
	}
}

// ESMappings sets a custom mapping from index name to geometry field name
func ESMappings(esMappings map[string]string) ConfigOption {
	return func(s *Server) error {
		s.ESMappings = esMappings
		return nil
	}
}

// ZoomRanges sets min and max zoom limits for a specific index
func ZoomRanges(strZoomRanges map[string]string) ConfigOption {
	return func(s *Server) error {
		zoomRanges := make(map[string][]int)
		for featureType, rangeStr := range strZoomRanges {
			minZoom := MinZoom
			maxZoom := MaxZoom
			zoomRangeParts := strings.Split(rangeStr, "-")
			if len(zoomRangeParts) >= 1 {
				minZoom, _ = strconv.Atoi(zoomRangeParts[0])
			}
			if len(zoomRangeParts) == 2 {
				maxZoom, _ = strconv.Atoi(zoomRangeParts[1])
			}
			zoomRanges[featureType] = []int{minZoom, maxZoom}
		}
		s.ZoomRanges = zoomRanges
		return nil
	}
}`
