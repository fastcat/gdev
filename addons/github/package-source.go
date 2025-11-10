package github

import (
	"fmt"
	"net/http"
	"strings"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

func PackageSource(
	ghc *Client,
	owner, repo, tag string,
	assetMatch func(*Release, *ReleaseAsset) bool,
) apt.PackageSource[*ghr] {
	if ghc == nil {
		ghc = NewClient()
	}
	return &apt.HTTPPackageSource[*ghr]{
		Rel: func(ctx *bootstrap.Context) (*ghr, error) {
			rel, err := ghc.GetRelease(ctx, owner, repo, tag)
			return (*ghr)(rel), err
		},
		Req: func(ctx *bootstrap.Context, rel *ghr) (*http.Request, error) {
			ai := -1
			for i := range rel.Assets {
				if assetMatch((*Release)(rel), &rel.Assets[i]) {
					ai = i
					break
				}
			}
			if ai < 0 {
				return nil, fmt.Errorf(
					"failed to find matching artifact in %s/%s release %s",
					owner, repo, rel.TagName,
				)
			}
			return ghc.DownloadReq(ctx, rel.Assets[ai].URL)
		},
	}
}

type ghr Release

var _ apt.PackageRelease = (*ghr)(nil)

func (r *ghr) PackageVersion() string {
	return strings.TrimPrefix(r.TagName, "v")
}
