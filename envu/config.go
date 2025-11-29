package envu

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

func MustLoadConfig[T any](overridePrefix string) T {
	c, err := LoadConfig[T](overridePrefix)
	if err != nil {
		panic(err)
	}
	return c
}

/*
Define a config struct like:

type Configuration struct {
	StripeSecretKey      string `env:"STRIPE_SECRET_KEY"`
	StripeWebhookSecret  string `env:"STRIPE_WEBHOOK_SECRET"`
	StripeProductPriceID string `env:"STRIPE_PRODUCT_PRICE_ID"`
	StripeRedirectDomain string `env:"STRIPE_REDIRECT_DOMAIN"`

	DiscordClientID     string `env:"DISCORD_CLIENT_ID"`
	DiscordClientSecret string `env:"DISCORD_CLIENT_SECRET"`
	DiscordAPIEndpoint  string `env:"DISCORD_API_ENDPOINT,https://discord.com/api/v10"`

	PostgresPort       string `env:"POSTGRES_PORT,5432"`
	PostgresHost       string `env:"POSTGRES_HOST,localhost"`
	PostgresPassword   string `env:"POSTGRES_PASSWORD"`
	PostgresSearchPath string `env:"POSTGRES_SEARCH_PATH,public"`
}

Using tags to define the env var name and potential default value.

Then load it with this function: cfg, err := LoadConfig[Configuration]("")


todo: make it work better for values that are slices
*/

func LoadConfig[T any](overridePrefix string) (T, error) {
	var config T
	v := reflect.ValueOf(&config).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		envTag := field.Tag.Get("env")
		if envTag == "" {
			continue
		}

		parts := strings.SplitN(envTag, ",", 2)
		envVarName := parts[0]

		// Get the string value from environment or default
		var value string
		if overridePrefix != "" {
			// if there is a value for the override prefix (like 'STAGING_') then use that.
			value = os.Getenv(overridePrefix + envVarName)
		}
		if value == "" {
			value = os.Getenv(envVarName)
			if value == "" && len(parts) > 1 {
				value = parts[1]
			} else if value == "" {
				return config, fmt.Errorf("env var %v missing", envVarName)
			}
		}

		// Decode the string value to the appropriate type
		switch field.Type.Kind() {
		case reflect.String:
			fieldValue.SetString(value)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			decoded, err := Decode[int64](value)
			if err != nil {
				return config, fmt.Errorf("failed to decode %v as int: %v", envVarName, err)
			}
			fieldValue.SetInt(decoded)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			decoded, err := Decode[uint64](value)
			if err != nil {
				return config, fmt.Errorf("failed to decode %v as uint: %v", envVarName, err)
			}
			fieldValue.SetUint(decoded)
		case reflect.Float32, reflect.Float64:
			decoded, err := Decode[float64](value)
			if err != nil {
				return config, fmt.Errorf("failed to decode %v as float: %v", envVarName, err)
			}
			fieldValue.SetFloat(decoded)
		case reflect.Bool:
			decoded, err := Decode[bool](value)
			if err != nil {
				return config, fmt.Errorf("failed to decode %v as bool: %v", envVarName, err)
			}
			fieldValue.SetBool(decoded)
		default:
			// For complex types (slices, structs, etc.), use reflection to decode
			// Create a new value of the field's type

			// Use Decode with the specific type
			result := reflect.ValueOf(Decode[any]).Call([]reflect.Value{reflect.ValueOf(value)})
			if !result[1].IsNil() {
				return config, fmt.Errorf("failed to decode %v: %v", envVarName, result[1].Interface().(error))
			}

			// Convert the result to the correct type and set it
			convertedValue := result[0].Convert(field.Type)
			fieldValue.Set(convertedValue)
		}
	}

	return config, nil
}
