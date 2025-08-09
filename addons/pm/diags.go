package pm

import (
	"context"
	"fmt"

	"fastcat.org/go/gdev/addons/diags"
	"fastcat.org/go/gdev/addons/pm/client"
)

func DiagsSources() diags.SourceProvider {
	return func(ctx context.Context) ([]diags.Source, error) {
		return []diags.Source{
			diags.SourceFunc(collectPMStatus),
			diags.SourceFunc(collectPMLogs),
		}, nil
	}
}

func collectPMStatus(ctx context.Context, coll diags.Collector) error {
	pmc := client.NewHTTP()
	if err := pmc.Ping(ctx); err != nil {
		// probably not running
		return coll.AddError(ctx, "pm/status.json", err)
	}
	stats, err := pmc.Summary(ctx)
	if err != nil {
		return coll.AddError(ctx, "pm/summary.json", err)
	}
	if err := diags.CollectJSON(ctx, coll, "pm/summary.json", stats); err != nil {
		return err
	}
	for _, c := range stats {
		cfn := "pm/child/" + c.Name + ".json"
		if details, err := pmc.Child(ctx, c.Name); err != nil {
			if err := coll.AddError(ctx, cfn, err); err != nil {
				return err
			}
			continue
		} else if err := diags.CollectJSON(ctx, coll, cfn, details); err != nil {
			return err
		}
	}
	return nil
}

func collectPMLogs(ctx context.Context, coll diags.Collector) error {
	// TODO: since pm service logs are collected by journald, use journalctl with
	// its JSON output mode to retrieve and process these. This is a bit
	// challenging since we need to split the journal stream up into each
	// invocation of each unit as separate collected pseudo-logfiles, and we need
	// to be sure we don't create deadlocks with the collection interface, so we
	// likely need to buffer each pseudo-log.

	return coll.AddError(ctx, "pm/logs.json", fmt.Errorf("pm doesn't capture logs yet"))
}
