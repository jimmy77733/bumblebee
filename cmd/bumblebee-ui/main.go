//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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
	light := a.Preferences().BoolWithFallback(prefLightTheme, false)
	a.Settings().SetTheme(newBumbleTheme(light))
	w := a.NewWindow("Bumblebee")
	w.Resize(fyne.NewSize(740, 560))

	catalogDir, catErr := scanrun.ResolveCatalogDir()
	lastReport := ""
	scanning := false
	checking := false
	hasCatalogUpdate := false
	statusBase := "就緒"
	var anim progressAnim

	title := widget.NewLabelWithStyle("Bumblebee", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	title.Importance = widget.HighImportance
	disclaimer := widget.NewLabel("僅比對已知曝光清單中的精確套件版本，不是 CVE 掃描，也不會讀取或執行原始碼。不會掃描整顆 C 槽。")
	disclaimer.Wrapping = fyne.TextWrapWord
	disclaimer.Importance = widget.LowImportance
	themeBtn := widget.NewButton(themeToggleLabel(light), nil)
	themeBtn.OnTapped = func() {
		light = !light
		a.Preferences().SetBool(prefLightTheme, light)
		a.Settings().SetTheme(newBumbleTheme(light))
		themeBtn.SetText(themeToggleLabel(light))
		w.Content().Refresh()
	}

	progress := newThinProgressBar()
	statusHead, statusPath, statusBox := newScanStatus()
	catalogStatus := widget.NewLabel("清單：檢查中…")
	catalogStatus.Wrapping = fyne.TextWrapWord

	viewBtn := widget.NewButton("檢視報告", nil)
	viewBtn.Disable()
	updateBtn := widget.NewButton("更新清單", nil)
	updateBtn.Disable()
	checkBtn := widget.NewButton("立即檢查更新", nil)
	dirBtn := widget.NewButton("掃描指定目錄", nil)
	smartBtn := widget.NewButton("智能掃描", nil)
	dirBtn.Importance = widget.HighImportance
	smartBtn.Importance = widget.HighImportance

	paintStatus := func(text string, imp widget.Importance) {
		statusBase = text
		head, path := splitStatusText(text)
		eta := anim.ETAText()
		if eta != "" {
			head = head + "　　" + eta
		}
		statusHead.Importance = imp
		statusPath.Importance = imp
		statusHead.SetText(head)
		statusPath.SetText(path)
		statusHead.Refresh()
		statusPath.Refresh()
	}

	refreshStatusLine := func() {
		paintStatus(statusBase, statusHead.Importance)
	}

	setScanBusy := func(busy bool) {
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
		if !checking {
			checkBtn.Enable()
		}
		if hasCatalogUpdate {
			updateBtn.Enable()
		}
		if lastReport != "" {
			viewBtn.Enable()
			viewBtn.Importance = widget.SuccessImportance
			viewBtn.Refresh()
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
				catalogStatus.Importance = widget.WarningImportance
				catalogStatus.Refresh()
				updateBtn.Disable()
				return
			}
		}
		text := "清單 " + catalogDate(st, catalogDir) + "（" + shortSHA(st.LocalSHA) + "）"
		hasCatalogUpdate = st.UpdateAvailable
		if st.UpdateAvailable {
			text += "　官方曝光清單有更新"
			catalogStatus.Importance = widget.WarningImportance
			updateBtn.Importance = widget.WarningImportance
			if !scanning {
				updateBtn.Enable()
			}
		} else {
			catalogStatus.Importance = widget.SuccessImportance
			updateBtn.Importance = widget.MediumImportance
			updateBtn.Disable()
		}
		catalogStatus.SetText(text)
		catalogStatus.Refresh()
		updateBtn.Refresh()
	}

	runCheck := func() {
		if catalogDir == "" || scanning || checking {
			return
		}
		checking = true
		checkBtn.Disable()
		anim.Start(8*time.Second, progress, refreshStatusLine)
		paintStatus("正在檢查官方清單…", widget.WarningImportance)
		go func() {
			st, err := catalogupdate.NewClient().Check(context.Background(), catalogDir)
			fyne.Do(func() {
				checking = false
				anim.Finish()
				refreshCatalog(st, err)
				if !scanning {
					checkBtn.Enable()
				}
				if err != nil {
					paintStatus("清單檢查失敗，沿用本機清單", widget.WarningImportance)
					return
				}
				if st.UpdateAvailable {
					paintStatus("官方曝光清單有更新", widget.WarningImportance)
					return
				}
				paintStatus("清單已是最新", widget.SuccessImportance)
			})
		}()
	}

	startScan := func(mode, profile string, scanRoots []scanner.Root, expected time.Duration) {
		if scanning {
			return
		}
		if catErr != nil {
			dialog.ShowError(catErr, w)
			return
		}
		checking = false
		setScanBusy(true)
		anim.Start(expected, progress, refreshStatusLine)
		paintStatus("開始掃描…", widget.WarningImportance)
		go func() {
			oc := scanrun.Run(context.Background(), scanrun.Options{
				Mode:       mode,
				Profile:    profile,
				Roots:      scanRoots,
				CatalogDir: catalogDir,
				OnProgress: func(p scanner.Progress) {
					fyne.Do(func() {
						msg := fmt.Sprintf("已檢查 %d 個檔案，曝光 %d 筆", p.FilesConsidered, p.FindingsEmitted)
						if p.CurrentPath != "" {
							msg += "\n" + p.CurrentPath
						}
						paintStatus(msg, widget.WarningImportance)
					})
				},
			})
			fyne.Do(func() {
				anim.Finish()
				setScanBusy(false)
				if oc.Report != "" {
					lastReport = oc.Report
					viewBtn.Enable()
					viewBtn.Importance = widget.SuccessImportance
					viewBtn.Refresh()
				}
				if oc.Err != nil && oc.Status == model.ScanStatusError {
					paintStatus("掃描失敗："+oc.Err.Error(), widget.DangerImportance)
					dialog.ShowError(oc.Err, w)
					return
				}
				imp := widget.SuccessImportance
				msg := fmt.Sprintf("掃描完成：%s，曝光 %d 筆，套件 %d 筆，檔案 %d", oc.Status, len(oc.Findings), oc.Result.RecordsEmitted, oc.Result.FilesConsidered)
				if len(oc.Findings) > 0 {
					imp = widget.WarningImportance
					msg = fmt.Sprintf("發現 %d 筆曝光 · %s · 套件 %d · 檔案 %d", len(oc.Findings), oc.Status, oc.Result.RecordsEmitted, oc.Result.FilesConsidered)
				}
				if oc.Err != nil {
					imp = widget.WarningImportance
					msg += "（" + oc.Err.Error() + "）"
				}
				paintStatus(msg, imp)
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
			startScan("指定目錄", model.ProfileDeep, r, 30*time.Second)
		}, w)
	}
	smartBtn.OnTapped = func() {
		r, _, err := scanrun.SmartRoots()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		startScan("智能掃描", model.ProfileDeep, r, 90*time.Second)
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
			setScanBusy(true)
			anim.Start(12*time.Second, progress, refreshStatusLine)
			paintStatus("正在下載官方清單…", widget.WarningImportance)
			go func() {
				aerr := catalogupdate.NewClient().Apply(context.Background(), catalogDir, st.RemoteSHA)
				fyne.Do(func() {
					anim.Finish()
					setScanBusy(false)
					if aerr != nil {
						paintStatus("更新清單失敗", widget.DangerImportance)
						dialog.ShowError(aerr, w)
						return
					}
					st.LocalSHA = st.RemoteSHA
					st.UpdateAvailable = false
					refreshCatalog(st, nil)
					paintStatus("已套用官方曝光清單 "+shortSHA(st.RemoteSHA), widget.SuccessImportance)
				})
			}()
		}, w)
	}
	checkBtn.OnTapped = func() {
		runCheck()
	}

	scanBlock := container.NewPadded(container.NewVBox(
		widget.NewLabelWithStyle("掃描", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(dirBtn, smartBtn, viewBtn),
		progress,
		statusBox,
	))
	catalogBlock := container.NewPadded(container.NewVBox(
		widget.NewLabelWithStyle("曝光清單", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		catalogStatus,
		container.NewHBox(updateBtn, checkBtn),
	))
	content := container.NewVBox(
		container.NewPadded(container.NewVBox(
			container.NewBorder(nil, nil, nil, themeBtn, title),
			disclaimer,
		)),
		widget.NewSeparator(),
		scanBlock,
		widget.NewSeparator(),
		catalogBlock,
	)
	w.SetContent(container.NewPadded(content))

	if catErr != nil {
		catalogStatus.SetText("找不到 threat_intel：" + catErr.Error())
		catalogStatus.Importance = widget.DangerImportance
		paintStatus("找不到 threat_intel", widget.DangerImportance)
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
