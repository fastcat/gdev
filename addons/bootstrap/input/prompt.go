package input

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"golang.org/x/sys/unix"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/internal"
	"fastcat.org/go/gdev/instance"
)

type Provider[T any] func(context.Context) (T, bool, error)

type Writer[T any] func(context.Context, T) error

// A Prompter is used to ensure a value is set in the bootstrap context,
// prompting the user if necessary.
//
// The general priority order it follows is:
//  1. If the context already has a valid value, use it
//  2. If any loader returns a valid value, use it
//  3. If no loader returned a valid value and any loader returned an error,
//     fail with all the errors joined
//  4. If any guesser returns a valid value, default to it but continue
//  5. If no guesser returned a valid value and any guesser returned an error,
//     fail with all the errors joined
//  6. Prompt the user to provide a value or confirm the guessed value
//  7. If any writers are set, write the value out with each of them,
//     returning any errors joined
type Prompter[T any] struct {
	key         internal.InfoKey[T]
	prompt      string
	description string
	help        string

	password bool

	loaders  []Provider[T]
	guessers []Provider[T]
	writers  []Writer[T]

	stringer  func(T) string
	parser    func(string) (T, error)
	validator func(T) error
}

type PrompterOpt[T any] func(*Prompter[T])

func TextPrompt(
	key internal.InfoKey[string],
	prompt string,
	opts ...PrompterOpt[string],
) *Prompter[string] {
	p := &Prompter[string]{
		key:      key,
		prompt:   prompt,
		stringer: strIdent,
		parser:   strIdentP,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func SecretPrompt(
	key internal.InfoKey[string],
	prompt string,
	opts ...PrompterOpt[string],
) *Prompter[string] {
	p := &Prompter[string]{
		key:      key,
		prompt:   prompt,
		password: true,
		stringer: strIdent,
		parser:   strIdentP,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// NewPrompter creates a new prompter for any type. opts must include at least
// [WithParser] and [WithStringer] or else it will panic.
//
// Use [TextPrompt] or [SecretPrompt] for strings to avoid boilerplate.
func NewPrompter[T any](
	key internal.InfoKey[T],
	prompt string,
	opts ...PrompterOpt[T],
) *Prompter[T] {
	p := &Prompter[T]{
		key:    key,
		prompt: prompt,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.parser == nil {
		panic(fmt.Errorf("no parser set for prompter"))
	}
	if p.stringer == nil {
		panic(fmt.Errorf("no stringer set for prompter"))
	}
	return p
}

func WithDescription(desc string) PrompterOpt[string] {
	return func(p *Prompter[string]) {
		p.description = desc
	}
}

func WithHelp(help string) PrompterOpt[string] {
	return func(p *Prompter[string]) {
		p.help = help
	}
}

func WithPassword() PrompterOpt[string] {
	return func(p *Prompter[string]) {
		p.password = true
	}
}

func WithLoaders[T any](loaders ...Provider[T]) PrompterOpt[T] {
	return func(p *Prompter[T]) {
		p.loaders = append(p.loaders, loaders...)
	}
}

func WithGuessers[T any](guessers ...Provider[T]) PrompterOpt[T] {
	return func(p *Prompter[T]) {
		p.guessers = append(p.guessers, guessers...)
	}
}

func WithWriters[T any](writers ...Writer[T]) PrompterOpt[T] {
	return func(p *Prompter[T]) {
		p.writers = append(p.writers, writers...)
	}
}

func WithStringer[T any](stringer func(T) string) PrompterOpt[T] {
	return func(p *Prompter[T]) {
		p.stringer = stringer
	}
}

func WithParser[T any](parser func(string) (T, error)) PrompterOpt[T] {
	return func(p *Prompter[T]) {
		p.parser = parser
	}
}

func WithValidator[T any](validator func(T) error) PrompterOpt[T] {
	return func(p *Prompter[T]) {
		p.validator = validator
	}
}

func strIdent(s string) string           { return s }
func strIdentP(s string) (string, error) { return s, nil }

func (p *Prompter[T]) _key() internal.AnyInfoKey {
	return p.key
}

func (p *Prompter[T]) init(ctx *internal.Context) (value T, ok, guessed bool, err error) {
	value, ok = internal.Get(ctx, p.key)
	if ok && p.validator != nil {
		if p.validator(value) == nil {
			return value, true, false, nil
		} else {
			ok = false
		}
	}
	// if we are forcing prompts, then make all loaders into guessers
	if os.Getenv(strings.ToUpper(instance.AppName())+"_FORCE_PROMPTS") == "true" {
		p.guessers = append(p.loaders, p.guessers...)
		p.loaders = nil
	}

	// if we don't have a valid value, try to load one from persistence
	if !ok {
		var errs []error
		for _, loader := range p.loaders {
			if lv, ok, err := loader(ctx); err != nil {
				errs = append(errs, err)
			} else if ok {
				if p.validator != nil {
					if err := p.validator(lv); err != nil {
						errs = append(errs, fmt.Errorf("loaded value is invalid: %w", err))
						continue
					}
				}
				// we have a valid value. caller is responsible for saving it if appropriate
				return lv, true, false, nil
			}
		}
		if len(errs) > 0 {
			// value is ~ zero(T) here
			return value, false, false, fmt.Errorf("failed to load value: %v", errors.Join(errs...))
		}
	}

	// if we still don't have a valid value, try to guess one
	if !ok {
		var errs []error
		for _, guesser := range p.guessers {
			var gv T
			var err error
			if gv, ok, err = guesser(ctx); err != nil {
				errs = append(errs, err)
			} else if ok {
				if p.validator != nil {
					if err := p.validator(gv); err != nil {
						errs = append(errs, fmt.Errorf("guessed value is invalid: %w", err))
						continue
					}
				}
				// we have a valid guessed value
				return gv, true, true, nil
			}
		}
		if !ok && len(errs) > 0 {
			return value, false, true, fmt.Errorf("failed to guess value: %v", errors.Join(errs...))
		}
	}
	return value, false, false, nil
}

func (p *Prompter[T]) field(ctx *internal.Context) (huh.Field, error) {
	value, ok, guessed, err := p.init(ctx)
	if err != nil {
		return nil, err
	} else if ok && !guessed {
		internal.Save(ctx, p.key, value)
		return nil, nil
	}

	var str string
	if ok {
		str = p.stringer(value)
	}

	helpActive := false
	i := huh.NewInput().
		Key(fmt.Sprintf("%s", p.key)).
		Title(p.prompt).
		Value(&str).
		DescriptionFunc(func() string {
			if helpActive {
				if p.description != "" {
					return p.description + "\n\n" + p.help
				}
				return p.help
			}
			return p.description
		}, &helpActive)
	i.Validate(p.validateString(&helpActive))
	if p.password {
		// while EchoModeNone is more unix-y, it tends to confuse non-graybeards
		i.EchoMode(huh.EchoModePassword)
	}

	if ok {
		// TODO: capture other guesses as suggestions?
		i.Suggestions([]string{str})
	}
	return i, nil
}

func (p *Prompter[T]) finishForm(
	ctx *internal.Context,
	fld huh.Field,
) error {
	str := fld.GetValue().(string)
	value, err := p.parser(str)
	if err != nil {
		// should be unreachable
		return fmt.Errorf("invalid value: %w", err)
	}
	if p.validator != nil {
		if err := p.validator(value); err != nil {
			// should be unreachable
			return fmt.Errorf("invalid value: %w", err)
		}
	}
	internal.Save(ctx, p.key, value)
	var errs []error
	for _, writer := range p.writers {
		if err := writer(ctx, value); err != nil {
			errs = append(errs, fmt.Errorf("error writing value: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (p *Prompter[T]) validateString(help *bool) func(s string) error {
	return func(s string) error {
		if s == "?" && p.help != "" {
			*help = true
			// this is a horrible hack to make it resize the display due to the description changing
			time.AfterFunc(time.Millisecond, func() { _ = unix.Kill(os.Getpid(), unix.SIGWINCH) })
			return errors.New("help provided")
		}
		v, err := p.parser(s)
		if err != nil {
			return err
		}
		if p.validator != nil {
			if err := p.validator(v); err != nil {
				return err
			}
		}
		return nil
	}
}

func (p *Prompter[T]) Run(ctx *internal.Context) error {
	return RunPrompts(ctx, p)
}

func (p *Prompter[T]) Sim(ctx *internal.Context) error {
	value, ok, guessed, err := p.init(ctx)
	if err != nil {
		return err
	} else if !ok {
		fmt.Printf("Would prompt for %s\n", p.prompt)
		return nil
	}
	// in a sim (dry run), assume the user would confirm the guess as far as the
	// in-memory storage
	internal.Save(ctx, p.key, value)
	if guessed {
		fmt.Printf("Would confirm guessed value for %s: %s\n", p.key, p.stringer(value))
	} else {
		fmt.Printf("Would use existing value for %s: %s\n", p.key, p.stringer(value))
	}
	return nil
}

type HuhPrompter interface {
	field(*internal.Context) (huh.Field, error)
	finishForm(*internal.Context, huh.Field) error
	Sim(*internal.Context) error
	_key() internal.AnyInfoKey
}

func RunPrompts(
	ctx *internal.Context,
	prompts ...HuhPrompter,
) error {
	if len(prompts) == 0 {
		return nil
	}
	fields := make([]huh.Field, 0, len(prompts))
	for _, p := range prompts {
		fld, err := p.field(ctx)
		if err != nil {
			return fmt.Errorf("error creating field: %w", err)
		} else if fld != nil {
			fields = append(fields, fld)
		}
	}
	if len(fields) == 0 {
		return nil
	}
	f := huh.NewForm(huh.NewGroup(fields...)).
		WithProgramOptions(tea.WithAltScreen())
	if err := f.RunWithContext(ctx); err != nil {
		return err
	}

	var errs []error
	for i, p := range prompts {
		if err := p.finishForm(ctx, fields[i]); err != nil {
			errs = append(errs, fmt.Errorf("error finishing form: %w", err))
		}
	}
	return errors.Join(errs...)
}

func SimPrompts(
	ctx *internal.Context,
	prompts ...HuhPrompter,
) error {
	for _, p := range prompts {
		if err := p.Sim(ctx); err != nil {
			return fmt.Errorf("error simulating prompt %s: %w", p._key(), err)
		}
	}
	return nil
}

func PromptStep(
	name string,
	prompts ...HuhPrompter,
) *bootstrap.Step {
	return bootstrap.NewStep(
		name,
		func(ctx *bootstrap.Context) error {
			return RunPrompts(ctx, prompts...)
		},
		bootstrap.SimFunc(func(ctx *bootstrap.Context) error {
			return SimPrompts(ctx, prompts...)
		}),
	)
}
