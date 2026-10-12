param([string]$Target)
$ErrorActionPreference = 'Continue'
$app = $null
foreach ($p in @('KWPS.Application','Word.Application')) {
  try { $app = New-Object -ComObject $p; break } catch {}
}
if (-not $app) { Write-Output 'COM_FAIL_NO_PROGID'; exit 1 }
try { $app.DisplayAlerts = 0 } catch {}
$doc = $null
try { $doc = $app.Documents.Open($Target) } catch { Write-Output ('COM_FAIL_OPEN: ' + $_.Exception.Message); exit 1 }
try { $doc.Repaginate() } catch {}
$cnt = $doc.TablesOfContents.Count
Write-Output ('TOC_COUNT: ' + $cnt)
if ($cnt -lt 1) { $doc.Close(0); Write-Output 'COM_FAIL_NO_TOC'; exit 1 }
$toc = $doc.TablesOfContents.Item(1)
$toc.Update() | Out-Null
$i = 0
foreach ($p in $toc.Range.Paragraphs) {
  $i++
  $txt = ''
  try { $txt = $p.Range.Text } catch {}
  $flat = ($txt -replace "[\r\n\t]", '~')
  $flat = $flat -replace "\|", '/'
  Write-Output ('TOCPARA|' + $i + '|' + $flat)
}
Write-Output ('PARA_COUNT: ' + $i)
$doc.Save()
$doc.Close(0)
if ($app.Documents.Count -eq 0) { $app.Quit() }
Write-Output 'COM_SAVED'