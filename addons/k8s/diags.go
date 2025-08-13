package k8s

import (
	"context"
	"path"

	"fastcat.org/go/gdev/addons/diags"
)

func DiagsSources() diags.SourceProvider {
	return func(ctx context.Context) ([]diags.Source, error) {
		client, err := NewClient()
		if err != nil {
			return nil, err
		}
		var sources []diags.Source
		for _, ns := range []Namespace{addon.Config.namespace, "kube-system"} {
			sources = append(sources,
				accessorSource(accStatefulSet, client, ns),
				accessorSource(accDeployment, client, ns),
				accessorSource(accService, client, ns),
				accessorSource(accConfigMap, client, ns),
				accessorSource(accPVC, client, ns),
				accessorSource(accPV, client, ns),
				accessorSource(accCronJob, client, ns),
				accessorSource(accBatchJob, client, ns),
				accessorSource(accPod, client, ns),
				// TODO: handle secrets specially, censor the actual data
				// secretsSource(client, ns),
			)
		}
		// non-namespaced objects, but pretend they are in kube-system
		sources = append(sources,
			accessorSource(accNode, client, "kube-system"),
		)
		return sources, nil
	}
}

func accessorSource[
	Client client[Resource, Apply],
	Resource any,
	Apply apply[Apply],
](
	acc accessor[Client, Resource, Apply],
	kc Interface,
	namespace Namespace,
) diags.SourceFunc {
	client := acc.getClient(kc, Namespace(namespace))
	return func(ctx context.Context, coll diags.Collector) error {
		return collectAccessor(ctx, coll, acc, client, namespace)
	}
}

func collectAccessor[
	Client client[Resource, Apply],
	Resource any,
	Apply apply[Apply],
](
	ctx context.Context,
	coll diags.Collector,
	acc accessor[Client, Resource, Apply],
	client Client,
	namespace Namespace,
) error {
	tm := acc.typ
	base := path.Join("k8s", string(namespace), tm.Kind)

	list, err := acc.list(ctx, client, listOpts(ctx))
	if err != nil {
		return coll.AddError(ctx, path.Join(base, "list.json"), err)
	}
	for i := range list {
		_, om := acc.resourceMeta(&list[i])
		// TODO: check tm and om.Namespace match what we got from the accessor
		if err := diags.CollectJSON(ctx, coll, path.Join(base, om.Name+".json"), &list[i]); err != nil {
			// if we get here it's fatal
			return err
		}
	}

	return nil
}
