// Cash Register JavaScript Application
class CashRegister {
    constructor() {
        this.currentTransaction = {
            items: [],
            paymentMethod: '',
            total: 0
        };
        this.selectedItemIndex = -1;
        this.selectedKisim = null;
        this.priceInput = '';
        this.quantityInput = '1';
        this.kisim = [];
        this.qrScanner = null;
        
        this.init();
    }
    
    async init() {
        await this.loadKisim();
        this.setupEventListeners();
        this.updateClock();
        this.startTransaction();
        
        // Update clock every second
        setInterval(() => this.updateClock(), 1000);
        
        this.log('Yazar kasa sistemi başlatıldı');
    }
    
    async loadKisim() {
        try {
            const response = await fetch('/api/kisim');
            const data = await response.json();
            this.kisim = data.kisim;
            this.log(`${this.kisim.length} kısım yüklendi`);
        } catch (error) {
            this.showError('Kısımlar yüklenemedi: ' + error.message);
        }
    }
    
    setupEventListeners() {
        // Kisim buttons
        document.querySelectorAll('.kisim-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const kisimId = parseInt(e.currentTarget.dataset.kisimId);
                const kisimName = e.currentTarget.dataset.kisimName;
                const taxRate = parseInt(e.currentTarget.dataset.taxRate);
                this.selectKisim(kisimId, kisimName, taxRate);
            });
        });
        
        // Numeric keypad for price entry
        document.querySelectorAll('.num-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const num = e.currentTarget.dataset.num;
                this.addToPriceInput(num);
            });
        });
        
        document.getElementById('clear-btn').addEventListener('click', () => {
            this.clearPriceInput();
        });
        
        // Add item button
        document.getElementById('add-item-btn').addEventListener('click', () => {
            this.addItem();
        });
        
        // Quantity controls
        document.getElementById('qty-plus').addEventListener('click', () => {
            this.adjustQuantity(1);
        });
        
        document.getElementById('qty-minus').addEventListener('click', () => {
            this.adjustQuantity(-1);
        });
        
        // Update quantity when quantity input changes
        document.getElementById('quantity-input').addEventListener('change', () => {
            if (this.selectedItemIndex >= 0) {
                this.updateSelectedItemQuantity();
            }
        });
        
        // Payment method buttons
        document.querySelectorAll('.payment-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const method = e.currentTarget.dataset.method;
                this.setPaymentMethod(method);
            });
        });
        
        // Action buttons
        document.getElementById('checkout-btn').addEventListener('click', () => {
            this.checkout();
        });
        
        document.getElementById('delete-item-btn').addEventListener('click', () => {
            this.deleteSelectedItem();
        });
        
        document.getElementById('cancel-btn').addEventListener('click', () => {
            this.cancelTransaction();
        });
        
        // Transaction item selection
        document.addEventListener('click', (e) => {
            if (e.target.closest('.transaction-item')) {
                const index = parseInt(e.target.closest('.transaction-item').dataset.index);
                this.selectTransactionItem(index);
            }
        });
        
        // Modal buttons
        document.getElementById('qr-cancel').addEventListener('click', () => {
            this.hideQRModal();
        });
        
        document.getElementById('receipt-cancel').addEventListener('click', () => {
            this.hideReceiptModal();
        });
        
        document.getElementById('receipt-confirm').addEventListener('click', () => {
            this.processTransaction();
        });
    }
    
    selectKisim(kisimId, kisimName, taxRate) {
        this.selectedKisim = { id: kisimId, name: kisimName, taxRate: taxRate };
        
        // Highlight selected kisim
        document.querySelectorAll('.kisim-btn').forEach(btn => {
            btn.classList.remove('ring-4', 'ring-white');
        });
        
        event.currentTarget.classList.add('ring-4', 'ring-white');
        
        // Enable add button if price is entered
        this.updateAddButton();
        
        this.log(`Kısım seçildi: ${kisimName}`);
    }
    
    addToPriceInput(num) {
        const priceInput = document.getElementById('price-input');
        const currentValue = priceInput.value;
        
        if (num === '.' && currentValue.includes('.')) {
            return; // Only one decimal point
        }
        
        if (currentValue.length < 8) { // Max 8 characters
            priceInput.value = currentValue + num;
            this.updateAddButton();
        }
    }
    
    clearPriceInput() {
        document.getElementById('price-input').value = '';
        this.updateAddButton();
    }
    
    updateAddButton() {
        const priceInput = document.getElementById('price-input');
        const addBtn = document.getElementById('add-item-btn');
        const price = parseFloat(priceInput.value);
        
        addBtn.disabled = !this.selectedKisim || !price || price <= 0;
    }
    
    adjustQuantity(delta) {
        const qtyInput = document.getElementById('quantity-input');
        let qty = parseInt(qtyInput.value) || 1;
        qty = Math.max(1, qty + delta);
        qtyInput.value = qty;
        
        // If an item is selected, update its quantity immediately
        if (this.selectedItemIndex >= 0) {
            this.updateSelectedItemQuantity();
        }
    }
    
    async startTransaction() {
        try {
            await fetch('/api/transaction/start', { method: 'POST' });
            this.log('Yeni işlem başlatıldı');
        } catch (error) {
            this.showError('İşlem başlatılamadı: ' + error.message);
        }
    }
    
    async addItem() {
        if (!this.selectedKisim) {
            this.showError('Kısım seçin!');
            return;
        }
        
        const priceInput = document.getElementById('price-input');
        const qtyInput = document.getElementById('quantity-input');
        const price = parseFloat(priceInput.value);
        const quantity = parseInt(qtyInput.value) || 1;
        
        if (!price || price <= 0) {
            this.showError('Geçerli fiyat girin!');
            return;
        }
        
        try {
            const response = await fetch('/api/transaction/add-item', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    kisim_id: this.selectedKisim.id,
                    kisim_name: this.selectedKisim.name,
                    unit_price: price,
                    tax_rate: this.selectedKisim.taxRate,
                    description: `₺${price.toFixed(2)}`
                })
            });
            
            const data = await response.json();
            if (data.success) {
                this.currentTransaction.items = data.items;
                
                // Set quantity if not 1
                if (quantity !== 1) {
                    const lastIndex = data.items.length - 1;
                    await this.setItemQuantity(lastIndex, quantity);
                }
                
                this.updateTransactionDisplay();
                this.clearPriceInput();
                this.log(`Ürün eklendi: ${this.selectedKisim.name} - ₺${price.toFixed(2)}`);
            } else {
                this.showError(data.message);
            }
        } catch (error) {
            this.showError('Ürün eklenemedi: ' + error.message);
        }
    }
    
    async setItemQuantity(itemIndex, quantity) {
        try {
            const response = await fetch('/api/transaction/set-quantity', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    item_index: itemIndex,
                    quantity: quantity
                })
            });
            
            const data = await response.json();
            if (data.success) {
                this.currentTransaction.items = data.items;
                this.updateTransactionDisplay();
            }
        } catch (error) {
            this.log('Miktar ayarlanamadı: ' + error.message);
        }
    }
    
    async updateSelectedItemQuantity() {
        if (this.selectedItemIndex === -1) {
            this.showError('Ürün seçin!');
            return;
        }
        
        const qtyInput = document.getElementById('quantity-input');
        const quantity = parseInt(qtyInput.value) || 1;
        
        if (quantity <= 0) {
            // Delete item if quantity is 0 or negative
            await this.deleteSelectedItem();
            return;
        }
        
        try {
            const response = await fetch('/api/transaction/set-quantity', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    item_index: this.selectedItemIndex,
                    quantity: quantity
                })
            });
            
            const data = await response.json();
            if (data.success) {
                this.currentTransaction.items = data.items;
                this.updateTransactionDisplay();
                this.log(`Miktar güncellendi: ${quantity}`);
            } else {
                this.showError(data.message);
            }
        } catch (error) {
            this.showError('Miktar güncellenemedi: ' + error.message);
        }
    }
    
    async setPaymentMethod(method) {
        try {
            // Ensure we have an active transaction
            if (!this.currentTransaction || this.currentTransaction.items.length === 0) {
                this.showError('Önce ürün ekleyin!');
                return;
            }
            
            const response = await fetch('/api/transaction/payment', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ payment_method: method })
            });
            
            const data = await response.json();
            if (data.success) {
                this.currentTransaction.paymentMethod = method;
                document.getElementById('payment-display').textContent = `Ödeme: ${method}`;
                
                // Update payment button styles
                document.querySelectorAll('.payment-btn').forEach(btn => {
                    btn.classList.remove('ring-4', 'ring-white');
                });
                
                if (method === 'Nakit') {
                    document.getElementById('payment-cash').classList.add('ring-4', 'ring-white');
                } else {
                    document.getElementById('payment-card').classList.add('ring-4', 'ring-white');
                }
                
                this.updateCheckoutButton();
                this.log(`Ödeme yöntemi: ${method}`);
            } else {
                this.showError(data.message);
            }
        } catch (error) {
            this.showError('Ödeme yöntemi ayarlanamadı: ' + error.message);
        }
    }
    
    async checkout() {
        if (this.currentTransaction.items.length === 0) {
            this.showError('Sepette ürün yok!');
            return;
        }
        
        if (!this.currentTransaction.paymentMethod) {
            this.showError('Ödeme yöntemi seçin!');
            return;
        }
        
        // Generate receipt preview
        try {
            const response = await fetch('/api/transaction/generate-receipt', {
                method: 'POST'
            });
            
            const data = await response.json();
            if (data.success) {
                this.showReceiptModal(data.receipt);
                this.log('Fiş önizleme oluşturuldu');
            } else {
                this.showError(data.message);
            }
        } catch (error) {
            this.showError('Fiş oluşturulamadı: ' + error.message);
        }
    }
    
    async processTransaction() {
        this.hideReceiptModal();
        
        // Check if standalone mode or needs QR scan
        const standaloneMode = document.querySelector('[data-standalone]') !== null;
        
        if (standaloneMode) {
            // Process without QR scan
            await this.submitTransaction('mock_ephemeral_key');
        } else {
            // Show QR scanner
            this.showQRModal();
        }
    }
    
    async submitTransaction(ephemeralKey) {
        try {
            this.log('İşlem gönderiliyor...');
            const response = await fetch('/api/transaction/process', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ ephemeral_key: ephemeralKey })
            });
            
            const data = await response.json();
            if (data.success) {
                this.showSuccess('İşlem başarıyla tamamlandı!');
                this.log('İşlem tamamlandı');
                this.resetTransaction();
            } else {
                this.showError('İşlem başarısız: ' + data.message);
                this.resetTransaction(); // Cancel transaction on error
            }
        } catch (error) {
            this.showError('İşlem hatası: ' + error.message);
            this.resetTransaction();
        }
    }
    
    async cancelTransaction() {
        try {
            await fetch('/api/transaction/cancel', { method: 'POST' });
            this.resetTransaction();
            this.log('İşlem iptal edildi');
        } catch (error) {
            this.showError('İptal edilemedi: ' + error.message);
        }
    }
    
    resetTransaction() {
        this.currentTransaction = {
            items: [],
            paymentMethod: '',
            total: 0
        };
        this.selectedItemIndex = -1;
        this.selectedKisim = null;
        this.updateTransactionDisplay();
        
        // Clear payment selection
        document.getElementById('payment-display').textContent = '';
        document.querySelectorAll('.payment-btn').forEach(btn => {
            btn.classList.remove('ring-4', 'ring-white');
        });
        
        // Clear kisim selection
        document.querySelectorAll('.kisim-btn').forEach(btn => {
            btn.classList.remove('ring-4', 'ring-white');
        });
        
        // Clear inputs
        this.clearPriceInput();
        document.getElementById('quantity-input').value = '1';
        
        this.updateCheckoutButton();
        this.startTransaction();
    }
    
    updateTransactionDisplay() {
        const container = document.getElementById('transaction-display');
        const totalElement = document.getElementById('total-display');
        const deleteBtn = document.getElementById('delete-item-btn');
        
        if (this.currentTransaction.items.length === 0) {
            container.innerHTML = '<div class="text-center text-gray-500 py-8 font-sans">Kısım seçin...</div>';
            totalElement.textContent = '₺0.00';
            this.currentTransaction.total = 0;
            deleteBtn.disabled = true;
        } else {
            let total = 0;
            const itemsHtml = this.currentTransaction.items.map((item, index) => {
                const itemTotal = item.quantity * item.unit_price;
                total += itemTotal;
                const isSelected = index === this.selectedItemIndex;
                const bgClass = isSelected ? 'bg-blue-900' : '';
                
                return `
                    <div class="transaction-item flex justify-between py-1 cursor-pointer hover:bg-gray-800 rounded px-2 ${bgClass}" data-index="${index}">
                        <span class="truncate flex-1">${item.kisim_name}</span>
                        <span class="w-16 text-center">${item.quantity}</span>
                        <span class="w-20 text-right">₺${itemTotal.toFixed(2)}</span>
                    </div>
                `;
            }).join('');
            
            container.innerHTML = itemsHtml;
            totalElement.textContent = `₺${total.toFixed(2)}`;
            this.currentTransaction.total = total;
            deleteBtn.disabled = this.selectedItemIndex === -1;
        }
        
        this.updateCheckoutButton();
    }
    
    selectTransactionItem(index) {
        this.selectedItemIndex = index;
        this.updateTransactionDisplay();
        this.log(`Seçili ürün: ${this.currentTransaction.items[index]?.kisim_name}`);
    }
    
    async deleteSelectedItem() {
        if (this.selectedItemIndex === -1) {
            this.showError('Silmek için ürün seçin!');
            return;
        }
        
        try {
            const response = await fetch('/api/transaction/set-quantity', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    item_index: this.selectedItemIndex,
                    quantity: 0 // Setting to 0 removes the item
                })
            });
            
            const data = await response.json();
            if (data.success) {
                this.currentTransaction.items = data.items;
                this.selectedItemIndex = -1;
                this.updateTransactionDisplay();
                this.log('Ürün silindi');
            } else {
                this.showError(data.message);
            }
        } catch (error) {
            this.showError('Ürün silinemedi: ' + error.message);
        }
    }
    
    selectItem(index) {
        this.selectedItemIndex = index;
        const item = this.currentTransaction.items[index];
        
        // Update quantity input
        document.getElementById('quantity-input').value = item.quantity;
        
        this.log(`Ürün seçildi: ${item.kisim_name}`);
    }
    
    updateCheckoutButton() {
        const btn = document.getElementById('checkout-btn');
        const canCheckout = this.currentTransaction.items.length > 0 && this.currentTransaction.paymentMethod;
        btn.disabled = !canCheckout;
    }
    
    showQRModal() {
        document.getElementById('qr-modal').classList.remove('hidden');
        this.initQRScanner();
    }
    
    hideQRModal() {
        document.getElementById('qr-modal').classList.add('hidden');
        if (this.qrScanner) {
            this.qrScanner.stop();
            this.qrScanner = null;
        }
    }
    
    async initQRScanner() {
        try {
            const videoElement = document.createElement('video');
            videoElement.style.width = '100%';
            videoElement.style.maxHeight = '300px';
            
            document.getElementById('qr-reader').innerHTML = '';
            document.getElementById('qr-reader').appendChild(videoElement);
            
            this.qrScanner = new QrScanner(videoElement, 
                (result) => {
                    this.log(`QR kod tarandı: ${result.data.substring(0, 20)}...`);
                    this.hideQRModal();
                    this.submitTransaction(result.data);
                },
                {
                    highlightScanRegion: true,
                    highlightCodeOutline: true,
                }
            );
            
            await this.qrScanner.start();
            this.log('QR tarayıcı başlatıldı');
        } catch (error) {
            this.showError('QR tarayıcı başlatılamadı: ' + error.message);
            this.hideQRModal();
            // Fallback to mock key for demo
            await this.submitTransaction('mock_ephemeral_key_fallback');
        }
    }
    
    showReceiptModal(receipt) {
        const modal = document.getElementById('receipt-modal');
        const content = document.getElementById('receipt-content');
        
        // Format receipt for display
        const receiptText = this.formatReceipt(receipt);
        content.textContent = receiptText;
        
        modal.classList.remove('hidden');
    }
    
    hideReceiptModal() {
        document.getElementById('receipt-modal').classList.add('hidden');
    }
    
    formatReceipt(receipt) {
        let text = '';
        text += '========================================\n';
        text += `         ${receipt.store_name}\n`;
        text += '========================================\n';
        text += `VKN: ${receipt.store_vkn}\n`;
        text += `${receipt.store_address}\n`;
        text += '========================================\n';
        text += `Tarih: ${new Date(receipt.timestamp).toLocaleString('tr-TR')}\n`;
        text += `İşlem No: ${receipt.transaction_id}\n`;
        text += `Fiş No: ${receipt.receipt_serial}\n`;
        text += '========================================\n\n';
        
        receipt.items.forEach(item => {
            text += `${item.kisim_name.padEnd(20)} ${item.quantity}x${item.unit_price.toFixed(2)} ₺${item.total_price.toFixed(2)}\n`;
        });
        
        text += '\n----------------------------------------\n';
        text += `KDV %10: ₺${receipt.tax_breakdown.tax_10_percent.tax_amount.toFixed(2)}\n`;
        text += `KDV %20: ₺${receipt.tax_breakdown.tax_20_percent.tax_amount.toFixed(2)}\n`;
        text += `Toplam KDV: ₺${receipt.tax_breakdown.total_tax.toFixed(2)}\n\n`;
        text += `GENEL TOPLAM: ₺${receipt.total_amount.toFixed(2)}\n`;
        text += `Ödeme: ${receipt.payment_method}\n`;
        text += '========================================\n';
        text += `Z Rapor No: ${receipt.z_report_number}\n`;
        text += '========================================\n';
        
        return text;
    }
    
    updateClock() {
        const now = new Date();
        const timeString = now.toLocaleTimeString('tr-TR');
        document.getElementById('current-time').textContent = timeString;
    }
    
    showSuccess(message) {
        this.showMessage(message, 'success');
    }
    
    showError(message) {
        this.showMessage(message, 'error');
        this.log(`HATA: ${message}`);
    }
    
    showMessage(message, type) {
        const container = document.getElementById('status-message');
        const bgColor = type === 'success' ? 'bg-green-500' : 'bg-red-500';
        
        container.innerHTML = `
            <div class="${bgColor} text-white px-4 py-3 rounded-lg shadow-lg">
                <div class="flex items-center">
                    <span>${message}</span>
                    <button onclick="this.parentElement.parentElement.parentElement.classList.add('hidden')" class="ml-4 text-white hover:text-gray-200">
                        ✕
                    </button>
                </div>
            </div>
        `;
        
        container.classList.remove('hidden');
        
        // Auto hide after 5 seconds
        setTimeout(() => {
            container.classList.add('hidden');
        }, 5000);
    }
    
    log(message) {
        const logContainer = document.getElementById('log-content');
        if (logContainer) {
            const timestamp = new Date().toLocaleTimeString('tr-TR');
            const logEntry = `[${timestamp}] ${message}\n`;
            logContainer.textContent += logEntry;
            logContainer.scrollTop = logContainer.scrollHeight;
        }
        console.log(`[CashRegister] ${message}`);
    }
}

// Initialize the cash register when page loads
let cashRegister;
document.addEventListener('DOMContentLoaded', () => {
    cashRegister = new CashRegister();
});