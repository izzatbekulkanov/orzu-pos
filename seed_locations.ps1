$items = @(
    @{ name = 'Coca-Cola 1.5L'; barcode = '5449000000996'; category_id = 5; buy_price = 11000; sell_price = 14000; stock_quantity = 120; unit = 'dona'; location = 'Muzlatgich №1' },
    @{ name = 'Validol 0.06g N20 (Yurak dorisi)'; barcode = '8690123456789'; category_id = 7; buy_price = 4500; sell_price = 7000; stock_quantity = 60; unit = 'quti'; location = 'Dorixona 2-polka' },
    @{ name = 'Samarqand noni (dona)'; barcode = '4780001234567'; category_id = 8; buy_price = 3500; sell_price = 5000; stock_quantity = 80; unit = 'dona'; location = '1-qator / Non bo''limi' },
    @{ name = 'Nestle Sut 3.2% 1L'; barcode = '4780018890012'; category_id = 9; buy_price = 12500; sell_price = 16000; stock_quantity = 50; unit = 'dona'; location = 'Muzlatgich №2' },
    @{ name = 'Qand (oq shakar 1kg)'; barcode = '4780029988771'; category_id = 6; buy_price = 9000; sell_price = 11500; stock_quantity = 200; unit = 'kg'; location = '3-qator / B-polka' },
    @{ name = 'Olma Beshyulduz (kg)'; barcode = '2000001002500'; category_id = 10; buy_price = 12000; sell_price = 16000; stock_quantity = 45.5; unit = 'kg'; location = 'Meva rastasi №1' }
)

foreach ($it in $items) {
    $json = $it | ConvertTo-Json
    Invoke-RestMethod -Uri 'http://localhost:8080/api/products' -Method POST -Body $json -ContentType 'application/json'
    Write-Host "Updated: $($it.name) with location: $($it.location)"
}
