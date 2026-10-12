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
$toc = $doc.TablesOfContents.Item(1)
$toc.Update() | Out-Null
try { Write-Output ('FIELDS_COUNT: ' + $toc.Range.Fields.Count) } catch { Write-Output 'FIELDS_COUNT: ERR' }
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
$doc.Close(0)
if ($app.Documents.Count -eq 0) { $app.Quit() }
Write-Output 'COM_DONE'