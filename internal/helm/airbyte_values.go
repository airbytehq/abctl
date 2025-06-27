package helm

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/maps"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ValuesOpts struct {
	ValuesFile      string
	LowResourceMode bool
	InsecureCookies bool
	TelemetryUser   string
	ImagePullSecret string
	DisableAuth     bool
	LocalStorage    bool
	EnablePsql17    bool
}

const (
	Psql17AirbyteTag = "1.7.0-17"
)

func BuildAirbyteValues(ctx context.Context, opts ValuesOpts) (string, error) {
	span := trace.SpanFromContext(ctx)

	vals := []string{
		"global.env_vars.AIRBYTE_INSTALLATION_ID=" + opts.TelemetryUser,
		"global.jobs.resources.limits.cpu=3",
		"global.jobs.resources.limits.memory=4Gi",
		"airbyte-bootloader.env_vars.PLATFORM_LOG_FORMAT=json",
	}

	if opts.LocalStorage {
		vals = append(vals, "global.storage.type=local")
	}

	if opts.EnablePsql17 {
		vals = append(vals, "postgresql.image.tag="+Psql17AirbyteTag)
	}

	span.SetAttributes(
		attribute.Bool("low-resource-mode", opts.LowResourceMode),
		attribute.Bool("insecure-cookies", opts.InsecureCookies),
		attribute.Bool("image-pull-secret", opts.ImagePullSecret != ""),
		attribute.Bool("local-storage", opts.LocalStorage),
	)

	if !opts.DisableAuth {
		vals = append(vals, "global.auth.enabled=true")
	}

	if opts.LowResourceMode {
		vals = append(vals,
			"server.env_vars.JOB_RESOURCE_VARIANT_OVERRIDE=lowresource",
			"global.jobs.resources.requests.cpu=0",
			"global.jobs.resources.requests.memory=0",

			"connector-builder-server.enabled=false",

			"workload-launcher.env_vars.CHECK_JOB_MAIN_CONTAINER_CPU_REQUEST=0",
			"workload-launcher.env_vars.CHECK_JOB_MAIN_CONTAINER_MEMORY_REQUEST=0",
			"workload-launcher.env_vars.DISCOVER_JOB_MAIN_CONTAINER_CPU_REQUEST=0",
			"workload-launcher.env_vars.DISCOVER_JOB_MAIN_CONTAINER_MEMORY_REQUEST=0",
			"workload-launcher.env_vars.SPEC_JOB_MAIN_CONTAINER_CPU_REQUEST=0",
			"workload-launcher.env_vars.SPEC_JOB_MAIN_CONTAINER_MEMORY_REQUEST=0",
			"workload-launcher.env_vars.SIDECAR_MAIN_CONTAINER_CPU_REQUEST=0",
			"workload-launcher.env_vars.SIDECAR_MAIN_CONTAINER_MEMORY_REQUEST=0",
		)
	}

	if opts.ImagePullSecret != "" {
		vals = append(vals, fmt.Sprintf("global.imagePullSecrets[0].name=%s", opts.ImagePullSecret))
	}

	if opts.InsecureCookies {
		vals = append(vals, "global.auth.cookieSecureSetting=false")
	}

	fileVals, err := maps.FromYAMLFile(opts.ValuesFile)
	if err != nil {
		return "", err
	}

	return mergeValuesWithValuesYAML(vals, fileVals)
}

// mergeValuesWithValuesYAML ensures that the values defined within this code have a lower
// priority than any values defined in a values.yaml file.
// By default, the helm-client we're using reversed this priority, putting the values
// defined in this code at a higher priority than the values defined in the values.yaml file.
// This function returns a string representation of the value.yaml file after all
// values provided were potentially overridden by the valuesYML file.
func mergeValuesWithValuesYAML(values []string, userValues map[string]any) (string, error) {
	a := maps.FromSlice(values)

	maps.Merge(a, userValues)

	res, err := maps.ToYAML(a)
	if err != nil {
		return "", fmt.Errorf("unable to merge values: %w", err)
	}

	return res, nil
}
