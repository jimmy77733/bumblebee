//go:build windows

package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/perplexityai/bumblebee/internal/catalogupdate"
	"github.com/perplexityai/bumblebee/internal/model"
	"github.com/perplexityai/bumblebee/internal/openurl"
	"github.com/perplexityai/bumblebee/internal/scanner"
	"github.com/perplexityai/bumblebee/internal/scanrun"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--check-catalog" {
		dir, err := scanrun.ResolveCatalogDir()
		if err == nil {
			_ = catalogupdate.RunSilentCheck(context.Background(), dir)
		}
		return
	}
	runUI()
}

func runUI() {
	a := app.NewWithID("bumblebee.windows.gui")
	w := a.NewWindow("Bumblebee")
	w.Resize(fyne.NewSize(680, 460))

	catalogDir, catErr := scanrun.ResolveCatalogDir()
	lastReport := ""
	scanning := false
	hasCatalogUpdate := false

	title := widget.NewLabelWithStyle("Bumblebee", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	disclaimer := widget.NewLabel("僅比對已知曝光清單中的精確套件版本，不是 CVE 掃描，也不會讀取或執行原始碼。不會掃描整顆 C 槽。")
	disclaimer.Wrapping = fyne.TextWrapWord

	progress := widget.NewProgressBar()
	status := widget.NewLabel("就緒")
	status.Wrapping = fyne.TextWrapWord
	catalogStatus := widget.NewLabel("清單：檢查中…")
	catalogStatus.Wrapping = fyne.TextWrapWord

	viewBtn := widget.NewButton("檢視報告", nil)
	viewBtn.Disable()
	updateBtn := widget.NewButton("更新清單", nil)
	updateBtn.Disable()
	checkBtn := widget.NewButton("立即檢查更新", nil)
	dirBtn := widget.NewButton("掃描指定目錄", nil)
	smartBtn := widget.NewButton("智能掃描", nil)

	setBusy := func(busy bool) {
		scanning = busy
		if busy {
			dirBtn.Disable()
			smartBtn.Disable()
			checkBtn.Disable()
			updateBtn.Disable()
			viewBtn.Disable()
			return
		}
		dirBtn.Enable()
		smartBtn.Enable()
		checkBtn.Enable()
		if hasCatalogUpdate {
			updateBtn.Enable()
		}
		if lastReport != "" {
			viewBtn.Enable()
		}
	}

	refreshCatalog := func(st catalogupdate.Status, err error) {
		if err != nil {
			if pending, ok := catalogupdate.ReadPending(); ok {
				st = pending
			} else {
				sha := catalogupdate.ReadLocalSHA(catalogDir)
				if sha == "" {
					catalogStatus.SetText("清單：離線，使用本機 threat_intel")
				} else {
					catalogStatus.SetText("清單：離線（" + shortSHA(sha) + "）")
				}
				updateBtn.Disable()
				return
			}
		}
		text := "清單 " + catalogDate(st, catalogDir) + "（" + shortSHA(st.LocalSHA) + "）"
		hasCatalogUpdate = st.UpdateAvailable
		if st.UpdateAvailable {
			text += "　官方曝光清單有更新"
			if !scanning {
				updateBtn.Enable()
			}
		} else {
			updateBtn.Disable()
		}
		catalogStatus.SetText(text)
	}

	runCheck := func() {
		if catalogDir == "" {
			return
		}
		go func() {
			st, err := catalogupdate.NewClient().Check(context.Background(), catalogDir)
			fyne.Do(func() { refreshCatalog(st, err) })
		}()
	}

	startScan := func(mode, profile string, scanRoots []scanner.Root) {
		if scanning {
			return
		}
		if catErr != nil {
			dialog.ShowError(catErr, w)
			return
		}
		setBusy(true)
		progress.SetValue(0)
		status.SetText("開始掃描…")
		go func() {
			oc := scanrun.Run(context.Background(), scanrun.Options{
				Mode:       mode,
				Profile:    profile,
				Roots:      scanRoots,
				CatalogDir: catalogDir,
				OnProgress: func(p scanner.Progress) {
					fyne.Do(func() {
						progress.SetValue(progressValue(p.FilesConsidered))
						msg := fmt.Sprintf("已檢查 %d 個檔案，曝光 %d 筆", p.FilesConsidered, p.FindingsEmitted)
						if p.CurrentPath != "" {
							msg += "\n" + p.CurrentPath
						}
						status.SetText(msg)
					})
				},
			})
			fyne.Do(func() {
				setBusy(false)
				if oc.Report != "" {
					lastReport = oc.Report
					viewBtn.Enable()
				}
				progress.SetValue(1)
				if oc.Err != nil && oc.Status == model.ScanStatusError {
					status.SetText("掃描失敗：" + oc.Err.Error())
					dialog.ShowError(oc.Err, w)
					return
				}
				msg := fmt.Sprintf("掃描完成：%s，曝光 %d 筆，套件 %d 筆，檔案 %d", oc.Status, len(oc.Findings), oc.Result.RecordsEmitted, oc.Result.FilesConsidered)
				if oc.Err != nil {
					msg += "（" + oc.Err.Error() + "）"
				}
				status.SetText(msg)
				if oc.Report != "" {
					_ = openurl.File(oc.Report)
				}
			})
		}()
	}

	dirBtn.OnTapped = func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				return
			}
			r, rerr := scanrun.SpecifiedDirRoots(uri.Path())
			if rerr != nil {
				dialog.ShowError(rerr, w)
				return
			}
			startScan("指定目錄", model.ProfileDeep, r)
		}, w)
	}
	smartBtn.OnTapped = func() {
		r, _, err := scanrun.SmartRoots()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		startScan("智能掃描", model.ProfileDeep, r)
	}
	viewBtn.OnTapped = func() {
		if lastReport == "" {
			return
		}
		if err := openurl.File(lastReport); err != nil {
			dialog.ShowError(err, w)
		}
	}
	updateBtn.OnTapped = func() {
		dialog.ShowConfirm("更新曝光清單", "將下載官方 threat_intel JSON 並覆蓋本機清單。掃描程式本身不會被更新。確定繼續？", func(ok bool) {
			if !ok {
				return
			}
			st, err := catalogupdate.NewClient().Check(context.Background(), catalogDir)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if !st.UpdateAvailable {
				refreshCatalog(st, nil)
				dialog.ShowInformation("清單", "目前已是官方最新清單。", w)
				return
			}
			setBusy(true)
			status.SetText("正在下載官方清單…")
			go func() {
				aerr := catalogupdate.NewClient().Apply(context.Background(), catalogDir, st.RemoteSHA)
				fyne.Do(func() {
					setBusy(false)
					if aerr != nil {
						status.SetText("更新清單失敗")
						dialog.ShowError(aerr, w)
						return
					}
					st.LocalSHA = st.RemoteSHA
					st.UpdateAvailable = false
					refreshCatalog(st, nil)
					status.SetText("已套用官方曝光清單 " + shortSHA(st.RemoteSHA))
				})
			}()
		}, w)
	}
	checkBtn.OnTapped = func() {
		status.SetText("正在檢查官方清單…")
		runCheck()
	}

	buttons := container.NewHBox(dirBtn, smartBtn, viewBtn)
	catalogRow := container.NewHBox(updateBtn, checkBtn)
	content := container.NewVBox(
		title,
		disclaimer,
		widget.NewSeparator(),
		buttons,
		progress,
		status,
		widget.NewSeparator(),
		catalogStatus,
		catalogRow,
		layout.NewSpacer(),
	)
	w.SetContent(container.NewPadded(content))

	if catErr != nil {
		catalogStatus.SetText("找不到 threat_intel：" + catErr.Error())
	} else {
		if exe, err := os.Executable(); err == nil {
			_ = catalogupdate.EnsureDailyTask(exe)
		}
		runCheck()
	}

	w.ShowAndRun()
}

func shortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) > 7 {
		return sha[:7]
	}
	if sha == "" {
		return "本機"
	}
	return sha
}

func catalogDate(st catalogupdate.Status, catalogDir string) string {
	if st.RemoteDate != "" {
		if tm, err := time.Parse(time.RFC3339, st.RemoteDate); err == nil {
			return tm.Format("2006-01-02")
		}
		if len(st.RemoteDate) >= 10 {
			return st.RemoteDate[:10]
		}
	}
	if info, err := os.Stat(catalogDir); err == nil {
		return info.ModTime().Format("2006-01-02")
	}
	return "未知日期"
}

func progressValue(files int) float64 {
	if files <= 0 {
		return 0.02
	}
	v := 1 - math.Exp(-float64(files)/800)
	if v > 0.97 {
		return 0.97
	}
	return v
}
