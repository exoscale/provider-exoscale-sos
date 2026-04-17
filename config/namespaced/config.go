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
}
