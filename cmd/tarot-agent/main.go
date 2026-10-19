package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/agents"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/bootstrap"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/cardlookup"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/daily"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/export"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/host/tui"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/update"
	"github.com/voocel/agentcore"
)

// version is the release version. GoReleaser overrides it with
// -X main.version={{.Version}}. Local builds stay "dev".
var version = "dev"

var (
	errFlagHelp  = errors.New("help")
	errFlagParse = errors.New("flag")
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, errFlagHelp) {
			os.Exit(0)
		}
		if errors.Is(err, errFlagParse) {
			os.Exit(2)
		}
		if errors.Is(err, update.ErrAvailable) {
			os.Exit(3)
		}
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

type cliOptions struct {
	mode        string
	showVersion bool
	checkUpdate bool
	checkOnly   bool
	yes         bool
	daily       bool
	update      bool
	export      bool
	exportArgs  []string
	card        bool
	cardArgs    []string
}

func run(args []string) error {
	opt, err := parseCLI(args)
	if err != nil {
		return err
	}
	ver := currentVersion()
	if opt.showVersion {
		fmt.Printf("tarot-agent %s\n", ver)
		return nil
	}
	if opt.update {
		return runUpdate(opt.checkOnly || opt.checkUpdate, opt.yes, ver)
	}
	if opt.checkUpdate || opt.checkOnly {
		return runCheckUpdate(ver)
	}
	if opt.daily {
		return runDaily(opt.mode)
	}
	if opt.export {
		return runExport(opt.exportArgs)
	}
	if opt.card {
		return runCard(opt.cardArgs)
	}
	return runTUI(opt.mode, ver)
}

// parseCLI accepts flags before or after daily and update.
// Any other positional argument is left alone so a plain launch stays the same.
func parseCLI(args []string) (cliOptions, error) {
	flagArgs, dailyCmd, updateCmd, exportCmd, exportArgs, cardCmd, cardArgs := extractCommands(args)
	fs := flag.NewFlagSet("tarot-agent", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	modeFlag := fs.String("mode", "", "reading mode: pro, professional, deep, casual, or simple")
	versionFlag := fs.Bool("version", false, "print version and exit")
	checkUpdate := fs.Bool("check-update", false, "compare with the latest GitHub release and exit")
	checkOnly := fs.Bool("check", false, "with update, compare only and do not install")
	yesLong := fs.Bool("yes", false, "update without asking")
	yesShort := fs.Bool("y", false, "update without asking")
	if perr := fs.Parse(flagArgs); perr != nil {
		if perr == flag.ErrHelp {
			return cliOptions{}, errFlagHelp
		}
		return cliOptions{}, errFlagParse
	}
	return cliOptions{
		mode:        *modeFlag,
		showVersion: *versionFlag,
		checkUpdate: *checkUpdate,
		checkOnly:   *checkOnly,
		yes:         *yesLong || *yesShort,
		daily:       dailyCmd,
		update:      updateCmd,
		export:      exportCmd,
		exportArgs:  exportArgs,
		card:        cardCmd,
		cardArgs:    cardArgs,
	}, nil
}

func extractCommands(args []string) (flagArgs []string, dailyCmd, updateCmd, exportCmd bool, exportArgs []string, cardCmd bool, cardArgs []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if isModeFlag(arg) {
			flagArgs = append(flagArgs, arg)
			if !strings.Contains(arg, "=") && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		switch arg {
		case "daily":
			if !dailyCmd {
				dailyCmd = true
				continue
			}
		case "update":
			if !updateCmd {
				updateCmd = true
				continue
			}
		case "export":
			if !exportCmd {
				return flagArgs, dailyCmd, updateCmd, true, append([]string{}, args[i+1:]...), cardCmd, cardArgs
			}
		case "card":
			if !cardCmd {
				return flagArgs, dailyCmd, updateCmd, exportCmd, exportArgs, true, append([]string{}, args[i+1:]...)
			}
		}
		flagArgs = append(flagArgs, arg)
	}
	return flagArgs, dailyCmd, updateCmd, exportCmd, exportArgs, cardCmd, cardArgs
}

func isModeFlag(arg string) bool {
	return arg == "--mode" || arg == "-mode" || strings.HasPrefix(arg, "--mode=") || strings.HasPrefix(arg, "-mode=")
}

func parseExportArgs(args []string) (export.Options, error) {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: tarot-agent export [id|latest] [--format md|txt] [-o 文件] [--force]\n")
		fmt.Fprintf(os.Stderr, "默认导出最近一条，格式 md，写到标准输出。-o 不覆盖已有文件，除非带 --force。\n")
		fs.PrintDefaults()
	}
	format := fs.String("format", "md", "md or txt")
	out := fs.String("o", "", "write to this file instead of stdout")
	force := fs.Bool("force", false, "replace an existing -o file")
	// The standard flag parser stops at the first id, so accept flags after it.
	if perr := fs.Parse(exportFlagsFirst(args)); perr != nil {
		if perr == flag.ErrHelp {
			return export.Options{}, errFlagHelp
		}
		return export.Options{}, errFlagParse
	}
	if len(fs.Args()) > 1 {
		return export.Options{}, fmt.Errorf("export 只接受一个记录 ID，或 latest")
	}
	query := "latest"
	if len(fs.Args()) == 1 {
		query = fs.Args()[0]
	}
	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "md", "markdown":
	case "txt", "text", "plain":
	default:
		return export.Options{}, fmt.Errorf("不认识的导出格式 %q，可用 md 或 txt", *format)
	}
	return export.Options{
		Query:      query,
		Format:     *format,
		OutputPath: *out,
		Force:      *force,
		Stdout:     os.Stdout,
	}, nil
}

func runExport(args []string) error {
	opt, err := parseExportArgs(args)
	if err != nil {
		return err
	}
	path, err := store.DefaultReadingsPath()
	if err != nil {
		return err
	}
	opt.Readings = store.NewReadingStore(path)
	if err := export.Execute(opt); err != nil {
		return fmt.Errorf("导出失败：%w", err)
	}
	return nil
}

func parseCardArgs(args []string) (query string, list bool, err error) {
	fs := flag.NewFlagSet("card", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: tarot-agent card <牌名> | tarot-agent card --list\n")
		fmt.Fprintf(os.Stderr, "离线查询本地牌库。支持中文名、英文名、常见别名和大阿卡那编号。不调用模型。\n")
		fs.PrintDefaults()
	}
	listFlag := fs.Bool("list", false, "list all 78 cards")
	if perr := fs.Parse(cardFlagsFirst(args)); perr != nil {
		if perr == flag.ErrHelp {
			return "", false, errFlagHelp
		}
		return "", false, errFlagParse
	}
	if *listFlag && len(fs.Args()) > 0 {
		return "", false, fmt.Errorf("card --list 不要再写牌名")
	}
	if !*listFlag && len(fs.Args()) == 0 {
		return "", false, fmt.Errorf("请写出牌名，或用 --list 列出全部")
	}
	return strings.Join(fs.Args(), " "), *listFlag, nil
}

func cardFlagsFirst(args []string) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			flags = append(flags, a)
			continue
		}
		positionals = append(positionals, a)
	}
	return append(flags, positionals...)
}

func runCard(args []string) error {
	query, list, err := parseCardArgs(args)
	if err != nil {
		return err
	}
	s, err := store.New()
	if err != nil {
		return fmt.Errorf("读取牌库: %w", err)
	}
	if err := cardlookup.Run(s.Cards.GetAll(), query, list, os.Stdout); err != nil {
		return fmt.Errorf("查牌失败：%w", err)
	}
	return nil
}

// exportFlagsFirst lets `export <id> --format txt` parse. Bool flags and
// flags with an attached value stay one token; the others take the next arg.
func exportFlagsFirst(args []string) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			positionals = append(positionals, a)
			continue
		}
		flags = append(flags, a)
		if strings.Contains(a, "=") || isExportBool(a) || isAttachedShort(a) {
			continue
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positionals...)
}

func isExportBool(arg string) bool {
	switch arg {
	case "-h", "--help", "--force", "-force":
		return true
	default:
		return false
	}
}

func isAttachedShort(arg string) bool {
	return strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 2
}

func runTUI(modeFlag, ver string) error {
	var cfg *bootstrap.Config

	// First-run setup
	if bootstrap.NeedsSetup() {
		var err error
		cfg, err = bootstrap.RunSetup()
		if err != nil {
			return fmt.Errorf("setup: %w", err)
		}
	} else {
		var err error
		cfg, err = bootstrap.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
	}

	mode, modeWarn := agents.ResolveMode(cfg.Mode, modeFlag)
	if modeWarn != "" {
		fmt.Fprintf(os.Stderr, "%s\n", modeWarn)
	}
	cfg.Mode = string(mode)

	// Initialize logger — write to file so it doesn't interfere with TUI
	logPath := bootstrap.LogFilePath()
	bootstrap.InitLogger(cfg.LogLevel, logPath)
	if modeWarn != "" {
		slog.Warn("invalid reading mode, using professional", "detail", modeWarn)
	}
	slog.Info("starting tarot-agent",
		"provider", cfg.Provider,
		"model", cfg.Model,
		"base_url", cfg.BaseURL,
	)

	// Initialize store
	s, err := store.New()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}
	slog.Info("store initialized",
		"cards", s.Cards.Count(),
		"spreads", len(s.Spreads.GetAll()),
	)

	// Initialize LLM model
	model, err := bootstrap.NewModel(cfg)
	if err != nil {
		return fmt.Errorf("init model: %w", err)
	}
	slog.Info("model initialized", "provider", cfg.Provider, "model", cfg.Model)

	// Build agent
	result := agents.BuildAgent(model, s, mode)
	slog.Info("agent built", "mode", mode.Label())

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		result.Agent.Abort()
	}()

	// Launch TUI. Version and daily return earlier, so they do not check for updates.
	return tui.Run(result.Agent, result.Guard, s, result.Mode.Label(), cfg.Model, ver, update.StartupEnabled(ver, os.Getenv))
}

func runCheckUpdate(ver string) error {
	ctx, cancel := context.WithTimeout(context.Background(), update.CheckTimeout)
	defer cancel()
	res, err := update.Check(ctx, nil, ver)
	if err != nil {
		return fmt.Errorf("检查更新失败：%w", err)
	}
	fmt.Print(update.FormatReport(res))
	return nil
}

// runUpdate installs the latest release. TAROT_AGENT_NO_UPDATE does not apply.
func runUpdate(checkOnly, yes bool, ver string) error {
	budget := update.UpdateTimeout
	if checkOnly {
		budget = update.CheckTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	err := update.Execute(ctx, update.ExecuteOptions{
		Current:    ver,
		CheckOnly:  checkOnly,
		Yes:        yes,
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		IsTerminal: isTerminal(os.Stdin),
	})
	if err == nil || errors.Is(err, update.ErrAvailable) {
		return err
	}
	if checkOnly {
		return fmt.Errorf("检查更新失败：%w", err)
	}
	return fmt.Errorf("更新失败：%w", err)
}

func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

func runDaily(modeFlag string) error {
	cfg, err := loadDailyConfig()
	if err != nil {
		return err
	}
	mode, modeWarn := agents.ResolveMode(cfg.Mode, modeFlag)
	if modeWarn != "" {
		fmt.Fprintf(os.Stderr, "%s\n", modeWarn)
	}

	s, err := store.New()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	var guide daily.Guide
	modelName := ""
	if cfg.APIKey != "" {
		modelName = cfg.Model
		model, merr := bootstrap.NewModelTimeout(cfg, daily.GuideTimeout)
		if merr != nil {
			guide = func(context.Context, string) (string, error) { return "", merr }
		} else {
			guide = func(ctx context.Context, prompt string) (string, error) {
				resp, gerr := model.Generate(ctx, []agentcore.Message{agentcore.UserMsg(prompt)}, nil)
				if gerr != nil {
					return "", gerr
				}
				if resp == nil {
					return "", fmt.Errorf("empty response")
				}
				return resp.Message.TextContent(), nil
			}
		}
	}
	return daily.Run(context.Background(), daily.Options{
		Store: s,
		Mode:  mode,
		Model: modelName,
		Out:   os.Stdout,
		Guide: guide,
	})
}

func loadDailyConfig() (*bootstrap.Config, error) {
	if bootstrap.NeedsSetup() {
		cfg := bootstrap.DefaultConfig()
		if v := os.Getenv("TAROT_MODE"); v != "" {
			cfg.Mode = v
		}
		cfg.APIKey = ""
		return cfg, nil
	}
	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}
