package github

import (
	"fmt"
	"net/http"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

func PackageSource(
	ghc *Client,
	owner, repo, tag string,
	assetMatch func(*Release, *ReleaseAsset) bool,
) apt.PackageSource {
	if ghc == nil {
		ghc = NewClient()
	}
	return apt.HTTPPackageSource(func(ctx *bootstrap.Context) (*http.Request, error) {
		rel, err := ghc.GetRelease(ctx, owner, repo, tag)
		if err != nil {
			return nil, err
		}
		ai := -1
		for i := range rel.Assets {
			if assetMatch(rel, &rel.Assets[i]) {
				ai = i
				break
			}
		}
		if ai < 0 {
			return nil, fmt.Errorf("failed to find matching artifact in %s/%s release %s", owner, repo, rel.TagName)
		}
		return ghc.DownloadReq(ctx, rel.Assets[ai].URL)
	})
}
