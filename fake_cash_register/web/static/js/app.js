// Cash Register JavaScript Application
class CashRegister {
    constructor() {
        this.currentTransaction = {
            items: [],
            paymentMethod: '',
            total: 0
        };
        this.currentInput = ''; // Current digit input being entered
        this.nextItemQuantity = 1; // Quantity captured by MIKTAR button
        this.inputMode = 'ambiguous'; // 'ambiguous', 'quantity', or 'price' mode
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
                const presetPrice = parseFloat(e.currentTarget.dataset.presetPrice);
                this.addKisimItem(kisimId, kisimName, taxRate, presetPrice);
            });
        });
        
        // Numeric keypad
        document.querySelectorAll('.num-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const num = e.currentTarget.dataset.num;
                this.addToCurrentInput(num);
            });
        });
        
        document.getElementById('clear-btn').addEventListener('click', () => {
            this.clearCurrentInput();
        });
        
        // MIKTAR button
        document.getElementById('miktar-btn').addEventListener('click', () => {
            this.captureMiktar();
        });
        
        // Payment method buttons - immediately complete transaction
        document.querySelectorAll('.payment-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const method = e.currentTarget.dataset.method;
                this.completeTransactionImmediately(method);
            });
        });
        
        // Action buttons
        document.getElementById('cancel-btn').addEventListener('click', () => {
            this.cancelTransaction();
        });
        
        
        // Modal buttons
        document.getElementById('qr-cancel').addEventListener('click', () => {
            this.hideQRModal();
        });
    }
    
    async addKisimItem(kisimId, kisimName, taxRate, presetPrice) {
        let finalPrice = presetPrice;
        let finalQuantity = this.nextItemQuantity;
        
        // Handle current input based on mode
        if (this.currentInput) {
            const inputValue = parseFloat(this.currentInput.replace(',', '.'));
            
            if (this.inputMode === 'price') {
                // Explicitly in price mode (after MIKTAR)
                finalPrice = inputValue > 0 ? inputValue : presetPrice;
            } else if (this.inputMode === 'ambiguous') {
                // Ambiguous mode: treat as price by default
                finalPrice = inputValue > 0 ? inputValue : presetPrice;
                finalQuantity = 1; // Reset quantity to 1 for custom price
            }
        }
        
        // Add the item with custom price and quantity
        await this.addNewItem(kisimId, finalQuantity, finalPrice);
        this.log(`Ürün eklendi: ${kisimName} - ₺${finalPrice.toFixed(2)} x${finalQuantity}`);
        
        // Reset state after adding item
        this.resetInputState();
    }
    
    addToCurrentInput(num) {
        // Convert period to comma for Turkish locale
        if (num === '.') {
            num = ',';
        }
        
        if (num === ',' && this.currentInput.includes(',')) {
            return; // Only one decimal separator
        }
        
        if (this.currentInput.length < 8) { // Max 8 characters
            this.currentInput += num;
            this.updateInputDisplay();
        }
    }
    
    clearCurrentInput() {
        this.currentInput = '';
        this.updateInputDisplay();
    }
    
    captureMiktar() {
        // Whatever is currently entered becomes the quantity
        const quantity = parseInt(this.currentInput) || 1;
        this.nextItemQuantity = Math.max(1, quantity);
        this.inputMode = 'price';
        this.currentInput = '';
        
        this.log(`MIKTAR: Sonraki ürün miktarı ${this.nextItemQuantity} olarak ayarlandı`);
        
        // Visual feedback to show MIKTAR was pressed
        const miktarBtn = document.getElementById('miktar-btn');
        miktarBtn.classList.add('ring-4', 'ring-yellow-300');
        setTimeout(() => {
            miktarBtn.classList.remove('ring-4', 'ring-yellow-300');
        }, 1000);
        
        this.updateInputDisplay();
    }
    
    resetInputState() {
        this.currentInput = '';
        this.nextItemQuantity = 1;
        this.inputMode = 'ambiguous';
        this.updateInputDisplay();
    }
    
    updateInputDisplay() {
        const inputElement = document.getElementById('current-input');
        if (this.currentInput) {
            let modeLabel = '';
            if (this.inputMode === 'price') {
                modeLabel = 'FİYAT: ';
            } else if (this.inputMode === 'quantity') {
                modeLabel = 'MİKTAR: ';
            } else {
                // ambiguous mode - show the number without label
                modeLabel = '';
            }
            inputElement.textContent = modeLabel + this.currentInput;
        } else {
            inputElement.textContent = '';
        }
    }
    
    async startTransaction() {
        try {
            const response = await fetch('/api/transaction/start', { method: 'POST' });
            if (response.ok) {
                this.log('Yeni işlem başlatıldı');
            } else {
                const errorData = await response.json();
                this.showError('İşlem başlatılamadı: ' + (errorData.error || 'Bilinmeyen hata'));
            }
        } catch (error) {
            this.showError('İşlem başlatılamadı: ' + error.message);
        }
    }
    
    async addNewItem(kisimId, quantity = 1, unitPrice = null) {
        try {
            const body = {
                kisim_id: kisimId,
                quantity: quantity
            };
            
            // Add custom unit price if provided
            if (unitPrice !== null && unitPrice > 0) {
                body.unit_price = unitPrice;
            }
            
            const response = await fetch('/api/transaction/add-item', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body)
            });
            
            if (response.ok) {
                const data = await response.json();
                this.currentTransaction.items = data.items;
                this.updateTransactionDisplay();
            } else {
                const errorData = await response.json();
                this.showError(errorData.error || 'Ürün eklenemedi');
            }
        } catch (error) {
            this.showError('Ürün eklenemedi: ' + error.message);
        }
    }
    
    // Remove this method since we now use addNewItem for everything
    // async addItemWithCustomQuantity - DEPRECATED
    
    async setItemQuantity(itemIndex, quantity) {
        try {
            // This endpoint doesn't exist in our API, so we'll skip this functionality
            // The CashRegister backend handles quantity through repeated add-item calls
            this.log('Miktar ayarlama - Backend tarafından otomatik olarak yönetiliyor');
        } catch (error) {
            this.log('Miktar ayarlanamadı: ' + error.message);
        }
    }
    
    async completeTransactionImmediately(method) {
        try {
            // Ensure we have an active transaction
            if (!this.currentTransaction || this.currentTransaction.items.length === 0) {
                this.showError('Önce ürün ekleyin!');
                return;
            }
            
            this.log(`Ödeme yöntemi: ${method} - İşlem tamamlanıyor...`);
            
            // Set payment method
            const paymentResponse = await fetch('/api/transaction/payment', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ payment_method: method })
            });
            
            if (!paymentResponse.ok) {
                const errorData = await paymentResponse.json();
                this.showError(errorData.error || 'Ödeme yöntemi ayarlanamadı');
                return;
            }
            
            // Check if standalone mode or needs QR scan
            const isStandaloneMode = document.body.dataset.standalone === 'true';
            
            if (isStandaloneMode) {
                // Process immediately without QR scan
                await this.submitTransaction('mock_ephemeral_key');
            } else {
                // Show QR scanner
                this.showQRModal();
            }
            
        } catch (error) {
            this.showError('İşlem tamamlanamadı: ' + error.message);
            this.resetTransaction();
        }
    }
    
    
    async submitTransaction(ephemeralKey) {
        try {
            this.log('İşlem gönderiliyor...');
            const response = await fetch('/api/transaction/issue_receipt', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ ephemeral_key: ephemeralKey })
            });
            
            if (response.ok) {
                const receipt = await response.json();
                this.showSuccess('İşlem başarıyla tamamlandı!');
                this.log(`İşlem tamamlandı - Fiş ID: ${receipt.receipt_id}`);
                this.resetTransaction();
            } else {
                const errorData = await response.json();
                this.showError('İşlem başarısız: ' + (errorData.error || 'Bilinmeyen hata'));
                this.resetTransaction(); // Cancel transaction on error
            }
        } catch (error) {
            this.showError('İşlem hatası: ' + error.message);
            this.resetTransaction();
        }
    }
    
    async cancelTransaction() {
        try {
            const response = await fetch('/api/transaction/cancel', { method: 'POST' });
            if (response.ok || response.status === 204) {
                this.resetTransaction();
                this.log('İşlem iptal edildi');
            } else {
                const errorData = await response.json();
                this.showError('İptal edilemedi: ' + (errorData.error || 'Bilinmeyen hata'));
            }
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
        this.resetInputState();
        this.updateTransactionDisplay();
        
        // Clear payment selection
        document.getElementById('payment-display').textContent = '';
        document.querySelectorAll('.payment-btn').forEach(btn => {
            btn.classList.remove('ring-4', 'ring-white');
        });
        
        this.startTransaction();
    }
    
    updateTransactionDisplay() {
        const container = document.getElementById('transaction-display');
        const totalElement = document.getElementById('total-display');
        
        if (this.currentTransaction.items.length === 0) {
            container.innerHTML = '<div class="text-center text-opacity-60 py-4 text-xs">0,00</div>';
            totalElement.textContent = '0,00';
            this.currentTransaction.total = 0;
        } else {
            let total = 0;
            const itemsHtml = this.currentTransaction.items.map((item, index) => {
                const itemTotal = item.quantity * item.unit_price;
                total += itemTotal;
                
                return `
                    <div class="flex justify-between text-xs py-1">
                        <span class="truncate">${item.kisim_name.substring(0, 8)}</span>
                        <span>${item.quantity}</span>
                        <span>${this.formatTurkishCurrency(itemTotal)}</span>
                    </div>
                `;
            }).join('');
            
            container.innerHTML = itemsHtml;
            totalElement.textContent = this.formatTurkishCurrency(total);
            this.currentTransaction.total = total;
        }
    }
    
    formatTurkishCurrency(amount) {
        // Format as Turkish currency with comma as decimal separator
        return amount.toFixed(2).replace('.', ',');
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
            
            // Create optimized canvas for better performance (eliminates Canvas2D warning)
            const canvas = document.createElement('canvas');
            canvas.getContext('2d', { willReadFrequently: true });
            
            document.getElementById('qr-reader').innerHTML = '';
            document.getElementById('qr-reader').appendChild(videoElement);
            
            this.qrScanner = new QrScanner(videoElement, 
                (result) => {
                    this.log(`QR kod tarandı: ${result.data.substring(0, 20)}...`);
                    this.hideQRModal();
                    this.submitTransaction(result.data);
                },
                {
                    returnDetailedScanResult: true,  // Fix: Enable modern API - result becomes {data, cornerPoints}
                    canvas: canvas,  // Use optimized canvas to eliminate performance warning
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
    
    
    updateClock() {
        const now = new Date();
        const timeString = now.toLocaleTimeString('tr-TR', {hour: '2-digit', minute: '2-digit', second: '2-digit'});
        const dateString = now.toLocaleDateString('tr-TR', {day: '2-digit', month: '2-digit', year: 'numeric'});
        
        const timeElement = document.getElementById('current-time');
        if (timeElement) {
            timeElement.textContent = timeString;
        }
        
        const dateElement = document.getElementById('current-date');
        if (dateElement) {
            dateElement.textContent = dateString;
        }
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