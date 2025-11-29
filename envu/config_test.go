package envu

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		overridePrefix string
		configType     interface{}
		want           interface{}
		wantErr        bool
		errContains    string
	}{
		{
			name: "all string fields with defaults",
			envVars: map[string]string{
				"APP_NAME": "myapp",
			},
			configType: struct {
				AppName string `env:"APP_NAME"`
				AppEnv  string `env:"APP_ENV,production"`
			}{},
			want: struct {
				AppName string `env:"APP_NAME"`
				AppEnv  string `env:"APP_ENV,production"`
			}{
				AppName: "myapp",
				AppEnv:  "production",
			},
			wantErr: false,
		},
		{
			name: "integer fields",
			envVars: map[string]string{
				"PORT":        "8080",
				"MAX_WORKERS": "10",
			},
			configType: struct {
				Port       int `env:"PORT"`
				MaxWorkers int `env:"MAX_WORKERS"`
				Timeout    int `env:"TIMEOUT,30"`
			}{},
			want: struct {
				Port       int `env:"PORT"`
				MaxWorkers int `env:"MAX_WORKERS"`
				Timeout    int `env:"TIMEOUT,30"`
			}{
				Port:       8080,
				MaxWorkers: 10,
				Timeout:    30,
			},
			wantErr: false,
		},
		{
			name: "boolean fields",
			envVars: map[string]string{
				"ENABLE_CACHE": "true",
				"DEBUG":        "false",
			},
			configType: struct {
				EnableCache bool `env:"ENABLE_CACHE"`
				Debug       bool `env:"DEBUG"`
				Verbose     bool `env:"VERBOSE,true"`
			}{},
			want: struct {
				EnableCache bool `env:"ENABLE_CACHE"`
				Debug       bool `env:"DEBUG"`
				Verbose     bool `env:"VERBOSE,true"`
			}{
				EnableCache: true,
				Debug:       false,
				Verbose:     true,
			},
			wantErr: false,
		},
		{
			name: "float fields",
			envVars: map[string]string{
				"RATE_LIMIT": "0.5",
				"THRESHOLD":  "99.99",
			},
			configType: struct {
				RateLimit float64 `env:"RATE_LIMIT"`
				Threshold float64 `env:"THRESHOLD"`
				Default   float64 `env:"DEFAULT,1.5"`
			}{},
			want: struct {
				RateLimit float64 `env:"RATE_LIMIT"`
				Threshold float64 `env:"THRESHOLD"`
				Default   float64 `env:"DEFAULT,1.5"`
			}{
				RateLimit: 0.5,
				Threshold: 99.99,
				Default:   1.5,
			},
			wantErr: false,
		},
		{
			name: "mixed types",
			envVars: map[string]string{
				"APP_NAME":   "testapp",
				"PORT":       "3000",
				"DEBUG":      "true",
				"MAX_CONNS":  "100",
				"RATE_LIMIT": "2.5",
			},
			configType: struct {
				AppName   string  `env:"APP_NAME"`
				Port      int     `env:"PORT"`
				Debug     bool    `env:"DEBUG"`
				MaxConns  uint    `env:"MAX_CONNS"`
				RateLimit float64 `env:"RATE_LIMIT"`
			}{},
			want: struct {
				AppName   string  `env:"APP_NAME"`
				Port      int     `env:"PORT"`
				Debug     bool    `env:"DEBUG"`
				MaxConns  uint    `env:"MAX_CONNS"`
				RateLimit float64 `env:"RATE_LIMIT"`
			}{
				AppName:   "testapp",
				Port:      3000,
				Debug:     true,
				MaxConns:  100,
				RateLimit: 2.5,
			},
			wantErr: false,
		},
		{
			name: "override prefix - staging values take precedence",
			envVars: map[string]string{
				"DB_HOST":         "prod.db.com",
				"STAGING_DB_HOST": "staging.db.com",
				"DB_PORT":         "5432",
			},
			overridePrefix: "STAGING_",
			configType: struct {
				DBHost string `env:"DB_HOST"`
				DBPort int    `env:"DB_PORT"`
			}{},
			want: struct {
				DBHost string `env:"DB_HOST"`
				DBPort int    `env:"DB_PORT"`
			}{
				DBHost: "staging.db.com",
				DBPort: 5432,
			},
			wantErr: false,
		},
		{
			name: "override prefix - falls back to normal when staging not set",
			envVars: map[string]string{
				"API_KEY": "production-key",
				"API_URL": "https://api.prod.com",
			},
			overridePrefix: "STAGING_",
			configType: struct {
				APIKey string `env:"API_KEY"`
				APIURL string `env:"API_URL"`
			}{},
			want: struct {
				APIKey string `env:"API_KEY"`
				APIURL string `env:"API_URL"`
			}{
				APIKey: "production-key",
				APIURL: "https://api.prod.com",
			},
			wantErr: false,
		},
		{
			name:    "missing required env var - no default",
			envVars: map[string]string{},
			configType: struct {
				RequiredField string `env:"REQUIRED_FIELD"`
			}{},
			wantErr:     true,
			errContains: "env var REQUIRED_FIELD missing",
		},
		{
			name: "invalid integer value",
			envVars: map[string]string{
				"PORT": "not-a-number",
			},
			configType: struct {
				Port int `env:"PORT"`
			}{},
			wantErr:     true,
			errContains: "failed to decode PORT as int",
		},
		{
			name: "invalid boolean value",
			envVars: map[string]string{
				"DEBUG": "not-a-bool",
			},
			configType: struct {
				Debug bool `env:"DEBUG"`
			}{},
			wantErr:     true,
			errContains: "failed to decode DEBUG as bool",
		},
		{
			name: "invalid float value",
			envVars: map[string]string{
				"RATE": "not-a-float",
			},
			configType: struct {
				Rate float64 `env:"RATE"`
			}{},
			wantErr:     true,
			errContains: "failed to decode RATE as float",
		},

		{
			name: "fields without env tag are skipped",
			envVars: map[string]string{
				"APP_NAME": "myapp",
			},
			configType: struct {
				AppName       string `env:"APP_NAME"`
				IgnoredField  string
				AnotherIgnore int
			}{},
			want: struct {
				AppName       string `env:"APP_NAME"`
				IgnoredField  string
				AnotherIgnore int
			}{
				AppName:       "myapp",
				IgnoredField:  "",
				AnotherIgnore: 0,
			},
			wantErr: false,
		},
		{
			name: "different integer sizes",
			envVars: map[string]string{
				"INT8":   "127",
				"INT16":  "32767",
				"INT32":  "2147483647",
				"UINT8":  "255",
				"UINT16": "65535",
			},
			configType: struct {
				Int8   int8   `env:"INT8"`
				Int16  int16  `env:"INT16"`
				Int32  int32  `env:"INT32"`
				Uint8  uint8  `env:"UINT8"`
				Uint16 uint16 `env:"UINT16"`
			}{},
			want: struct {
				Int8   int8   `env:"INT8"`
				Int16  int16  `env:"INT16"`
				Int32  int32  `env:"INT32"`
				Uint8  uint8  `env:"UINT8"`
				Uint16 uint16 `env:"UINT16"`
			}{
				Int8:   127,
				Int16:  32767,
				Int32:  2147483647,
				Uint8:  255,
				Uint16: 65535,
			},
			wantErr: false,
		},
		{
			name: "empty string default",
			envVars: map[string]string{
				"APP_NAME": "myapp",
			},
			configType: struct {
				AppName     string `env:"APP_NAME"`
				OptionalURL string `env:"OPTIONAL_URL,"`
			}{},
			want: struct {
				AppName     string `env:"APP_NAME"`
				OptionalURL string `env:"OPTIONAL_URL,"`
			}{
				AppName:     "myapp",
				OptionalURL: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set up environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Clean up after test
			defer os.Clearenv()

			// Call the appropriate LoadConfig variant based on type
			var got interface{}
			var err error

			// Use type assertion to call the correct LoadConfig variant
			switch tt.configType.(type) {
			case struct {
				AppName string `env:"APP_NAME"`
				AppEnv  string `env:"APP_ENV,production"`
			}:
				result, e := LoadConfig[struct {
					AppName string `env:"APP_NAME"`
					AppEnv  string `env:"APP_ENV,production"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				Port       int `env:"PORT"`
				MaxWorkers int `env:"MAX_WORKERS"`
				Timeout    int `env:"TIMEOUT,30"`
			}:
				result, e := LoadConfig[struct {
					Port       int `env:"PORT"`
					MaxWorkers int `env:"MAX_WORKERS"`
					Timeout    int `env:"TIMEOUT,30"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				EnableCache bool `env:"ENABLE_CACHE"`
				Debug       bool `env:"DEBUG"`
				Verbose     bool `env:"VERBOSE,true"`
			}:
				result, e := LoadConfig[struct {
					EnableCache bool `env:"ENABLE_CACHE"`
					Debug       bool `env:"DEBUG"`
					Verbose     bool `env:"VERBOSE,true"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				RateLimit float64 `env:"RATE_LIMIT"`
				Threshold float64 `env:"THRESHOLD"`
				Default   float64 `env:"DEFAULT,1.5"`
			}:
				result, e := LoadConfig[struct {
					RateLimit float64 `env:"RATE_LIMIT"`
					Threshold float64 `env:"THRESHOLD"`
					Default   float64 `env:"DEFAULT,1.5"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				AllowedHosts []string `env:"ALLOWED_HOSTS"`
				Ports        []int    `env:"PORTS"`
			}:
				result, e := LoadConfig[struct {
					AllowedHosts []string `env:"ALLOWED_HOSTS"`
					Ports        []int    `env:"PORTS"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				AppName   string  `env:"APP_NAME"`
				Port      int     `env:"PORT"`
				Debug     bool    `env:"DEBUG"`
				MaxConns  uint    `env:"MAX_CONNS"`
				RateLimit float64 `env:"RATE_LIMIT"`
			}:
				result, e := LoadConfig[struct {
					AppName   string  `env:"APP_NAME"`
					Port      int     `env:"PORT"`
					Debug     bool    `env:"DEBUG"`
					MaxConns  uint    `env:"MAX_CONNS"`
					RateLimit float64 `env:"RATE_LIMIT"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				DBHost string `env:"DB_HOST"`
				DBPort int    `env:"DB_PORT"`
			}:
				result, e := LoadConfig[struct {
					DBHost string `env:"DB_HOST"`
					DBPort int    `env:"DB_PORT"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				APIKey string `env:"API_KEY"`
				APIURL string `env:"API_URL"`
			}:
				result, e := LoadConfig[struct {
					APIKey string `env:"API_KEY"`
					APIURL string `env:"API_URL"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				RequiredField string `env:"REQUIRED_FIELD"`
			}:
				result, e := LoadConfig[struct {
					RequiredField string `env:"REQUIRED_FIELD"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				Port int `env:"PORT"`
			}:
				result, e := LoadConfig[struct {
					Port int `env:"PORT"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				Debug bool `env:"DEBUG"`
			}:
				result, e := LoadConfig[struct {
					Debug bool `env:"DEBUG"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				Rate float64 `env:"RATE"`
			}:
				result, e := LoadConfig[struct {
					Rate float64 `env:"RATE"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				Hosts []string `env:"HOSTS"`
			}:
				result, e := LoadConfig[struct {
					Hosts []string `env:"HOSTS"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				AppName       string `env:"APP_NAME"`
				IgnoredField  string
				AnotherIgnore int
			}:
				result, e := LoadConfig[struct {
					AppName       string `env:"APP_NAME"`
					IgnoredField  string
					AnotherIgnore int
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				Int8   int8   `env:"INT8"`
				Int16  int16  `env:"INT16"`
				Int32  int32  `env:"INT32"`
				Uint8  uint8  `env:"UINT8"`
				Uint16 uint16 `env:"UINT16"`
			}:
				result, e := LoadConfig[struct {
					Int8   int8   `env:"INT8"`
					Int16  int16  `env:"INT16"`
					Int32  int32  `env:"INT32"`
					Uint8  uint8  `env:"UINT8"`
					Uint16 uint16 `env:"UINT16"`
				}](tt.overridePrefix)
				got, err = result, e
			case struct {
				AppName     string `env:"APP_NAME"`
				OptionalURL string `env:"OPTIONAL_URL,"`
			}:
				result, e := LoadConfig[struct {
					AppName     string `env:"APP_NAME"`
					OptionalURL string `env:"OPTIONAL_URL,"`
				}](tt.overridePrefix)
				got, err = result, e
			default:
				t.Fatalf("unhandled config type in test")
			}

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && err != nil {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("LoadConfig() error = %v, should contain %v", err, tt.errContains)
					}
				}
				return
			}

			// Compare results
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
