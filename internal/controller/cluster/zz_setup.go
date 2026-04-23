// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	providerconfig "github.com/exoscale/provider-exoscale-sos/internal/controller/cluster/providerconfig"
	bucket "github.com/exoscale/provider-exoscale-sos/internal/controller/cluster/sos/bucket"
	bucketacl "github.com/exoscale/provider-exoscale-sos/internal/controller/cluster/sos/bucketacl"
	bucketcorsconfiguration "github.com/exoscale/provider-exoscale-sos/internal/controller/cluster/sos/bucketcorsconfiguration"
	bucketversioning "github.com/exoscale/provider-exoscale-sos/internal/controller/cluster/sos/bucketversioning"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		providerconfig.Setup,
		bucket.Setup,
		bucketacl.Setup,
		bucketcorsconfiguration.Setup,
		bucketversioning.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupGated creates all controllers with the supplied logger and adds them to
// the supplied manager gated.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		providerconfig.SetupGated,
		bucket.SetupGated,
		bucketacl.SetupGated,
		bucketcorsconfiguration.SetupGated,
		bucketversioning.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
