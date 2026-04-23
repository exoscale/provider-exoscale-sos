package namespaced

import "github.com/crossplane/upjet/v2/pkg/config"

const shortGroup = "sos"

// Configure configures individual resources by adding custom ResourceConfigurators.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("aws_s3_bucket", func(r *config.Resource) {
		r.Kind = "Bucket"
		r.ShortGroup = shortGroup

		config.MoveToStatus(r.TerraformResource, "acceleration_status", "acl", "grant", "cors_rule", "lifecycle_rule",
			"logging", "object_lock_configuration", "policy", "replication_configuration", "request_payer",
			"server_side_encryption_configuration", "versioning", "website", "arn", "bucket_namespace", "force_destroy", "region", "tags", "tags_all")
	})
	p.AddResourceConfigurator("aws_s3_bucket_versioning", func(r *config.Resource) {
		r.Kind = "BucketVersioning"
		r.ShortGroup = shortGroup

		r.References["bucket"] = config.Reference{
			Type: "github.com/exoscale/provider-exoscale-sos/apis/namespaced/sos/v1alpha1.Bucket",
		}
	})
}
