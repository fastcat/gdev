package bootstrap

import (
	"fastcat.org/go/gdev/internal"
)

func Shell(
	ctx *Context,
	cmdAndArgs []string,
	opts ...internal.ShellOpt,
) error {
	return internal.Shell(ctx, cmdAndArgs, opts...)
}

func WithSudo(purpose string) internal.ShellOpt {
	return internal.WithSudo(purpose)
}

func WithPassStdio() internal.ShellOpt {
	return internal.WithPassStdio()
}
