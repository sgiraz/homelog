# 💰 HomeLog - Split & Settlement Feature: Specifica Tecnica Completa

## 📋 INDICE
1. [Overview](#overview)
2. [Database Schema](#database-schema)
3. [Business Logic](#business-logic)
4. [API Endpoints](#api-endpoints)
5. [Frontend Components](#frontend-components)
6. [User Flow](#user-flow)
7. [Test Cases](#test-cases)
8. [Implementation Steps](#implementation-steps)

---

## 🎯 OVERVIEW

### Problema
Le famiglie gestiscono le spese in modi diversi:
- **Split Mode (Simone & Valentina)**: Ogni spesa viene divisa 50/50, si tiene traccia di chi deve cosa, si fanno bonifici per pareggiare
- **Shared Mode**: Spese condivise senza debiti, solo per tracking e analytics

### Soluzione
Sistema flessibile che permette di:
1. **Attivare/disattivare** split mode tramite impostazioni
2. **Tracciare chi paga** ogni spesa
3. **Calcolare automaticamente** chi deve cosa
4. **Registrare pagamenti** (settlements) per azzerare debiti
5. **Visualizzare bilancio** in tempo reale

---

## 🗄️ DATABASE SCHEMA

### 1. Modello `HouseholdSettings` (NUOVO)
```go
type HouseholdSettings struct {
    ID        uint      `gorm:"primarykey" json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    
    PropertyID      uint   `gorm:"uniqueIndex;not null" json:"property_id"` // FK to Property
    SplitMode       bool   `gorm:"not null;default:false" json:"split_mode"` // true = split, false = shared
    DefaultSplitType string `gorm:"not null;default:'equal'" json:"default_split_type"` // 'equal', 'custom', 'income_based'
    
    // Relations
    Property Property `json:"property"`
}
```

### 2. Modello `Expense` (MODIFICATO - aggiungi questi campi)
```go
type Expense struct {
    // ... campi esistenti ...
    
    // NUOVI CAMPI per Split
    PaidByUserID  uint   `gorm:"not null;index" json:"paid_by_user_id"` // Chi ha pagato fisicamente
    IsSplit       bool   `gorm:"not null;default:false" json:"is_split"` // Se la spesa è divisa
    
    // Relations
    PaidBy User         `gorm:"foreignKey:PaidByUserID" json:"paid_by"`
    Splits []ExpenseSplit `gorm:"foreignKey:ExpenseID" json:"splits,omitempty"`
}
```

### 3. Modello `ExpenseSplit` (NUOVO)
```go
type ExpenseSplit struct {
    ID        uint      `gorm:"primarykey" json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    
    ExpenseID   uint    `gorm:"not null;index" json:"expense_id"` // FK to Expense
    UserID      uint    `gorm:"not null;index" json:"user_id"`    // Chi deve questa quota
    Amount      float64 `gorm:"not null" json:"amount"`           // Importo quota
    IsSettled   bool    `gorm:"not null;default:false" json:"is_settled"` // Se è stata saldata
    SettledAt   *time.Time `json:"settled_at,omitempty"`
    SettlementID *uint  `gorm:"index" json:"settlement_id,omitempty"` // FK to Settlement (optional)
    
    // Relations
    Expense    Expense    `json:"expense"`
    User       User       `json:"user"`
    Settlement *Settlement `json:"settlement,omitempty"`
}
```

### 4. Modello `Settlement` (NUOVO)
```go
type Settlement struct {
    ID        uint           `gorm:"primarykey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
    
    PropertyID  uint       `gorm:"not null;index" json:"property_id"` // FK to Property
    FromUserID  uint       `gorm:"not null;index" json:"from_user_id"` // Chi paga
    ToUserID    uint       `gorm:"not null;index" json:"to_user_id"`   // Chi riceve
    Amount      float64    `gorm:"not null" json:"amount"`
    Date        time.Time  `gorm:"not null;index" json:"date"`
    PaymentMethod string   `json:"payment_method,omitempty"` // 'bank_transfer', 'cash', 'satispay', etc.
    Note        string     `json:"note,omitempty"`
    
    // Relations
    Property Property `json:"property"`
    FromUser User     `gorm:"foreignKey:FromUserID" json:"from_user"`
    ToUser   User     `gorm:"foreignKey:ToUserID" json:"to_user"`
    ExpenseSplits []ExpenseSplit `gorm:"foreignKey:SettlementID" json:"expense_splits,omitempty"`
}
```

### 5. Migration SQL
```sql
-- Aggiungi colonne a expenses
ALTER TABLE expenses ADD COLUMN paid_by_user_id INTEGER NOT NULL DEFAULT 1;
ALTER TABLE expenses ADD COLUMN is_split BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_expenses_paid_by_user_id ON expenses(paid_by_user_id);

-- Crea tabella household_settings
CREATE TABLE household_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    property_id INTEGER NOT NULL UNIQUE,
    split_mode BOOLEAN NOT NULL DEFAULT FALSE,
    default_split_type VARCHAR(20) NOT NULL DEFAULT 'equal',
    FOREIGN KEY (property_id) REFERENCES properties(id)
);

-- Crea tabella expense_splits
CREATE TABLE expense_splits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    expense_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    is_settled BOOLEAN NOT NULL DEFAULT FALSE,
    settled_at DATETIME,
    settlement_id INTEGER,
    FOREIGN KEY (expense_id) REFERENCES expenses(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (settlement_id) REFERENCES settlements(id)
);
CREATE INDEX idx_expense_splits_expense_id ON expense_splits(expense_id);
CREATE INDEX idx_expense_splits_user_id ON expense_splits(user_id);
CREATE INDEX idx_expense_splits_settlement_id ON expense_splits(settlement_id);

-- Crea tabella settlements
CREATE TABLE settlements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    property_id INTEGER NOT NULL,
    from_user_id INTEGER NOT NULL,
    to_user_id INTEGER NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    date DATE NOT NULL,
    payment_method VARCHAR(50),
    note TEXT,
    FOREIGN KEY (property_id) REFERENCES properties(id),
    FOREIGN KEY (from_user_id) REFERENCES users(id),
    FOREIGN KEY (to_user_id) REFERENCES users(id)
);
CREATE INDEX idx_settlements_property_id ON settlements(property_id);
CREATE INDEX idx_settlements_from_user_id ON settlements(from_user_id);
CREATE INDEX idx_settlements_to_user_id ON settlements(to_user_id);
CREATE INDEX idx_settlements_date ON settlements(date);
```

---

## 🧮 BUSINESS LOGIC

### Algoritmo Calcolo Bilancio

```go
// CalculateBalance calcola il bilancio tra currentUser e otherUser
func CalculateBalance(currentUserID, otherUserID, propertyID uint, db *gorm.DB) (float64, error) {
    var balance float64 = 0.0
    
    // 1. Verifica se split mode è attivo
    var settings HouseholdSettings
    if err := db.Where("property_id = ?", propertyID).First(&settings).Error; err != nil {
        return 0.0, err
    }
    
    if !settings.SplitMode {
        return 0.0, nil // Split mode disattivato = bilancio sempre 0
    }
    
    // 2. Recupera tutti gli split non saldati della property
    var splits []ExpenseSplit
    db.Where("is_settled = ? AND user_id IN (?)", false, []uint{currentUserID, otherUserID}).
       Preload("Expense").
       Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
       Where("expenses.property_id = ?", propertyID).
       Find(&splits)
    
    // 3. Calcola bilancio dagli split
    for _, split := range splits {
        if split.Expense.PaidByUserID == currentUserID && split.UserID == otherUserID {
            // Io ho pagato, other mi deve
            balance += split.Amount
        } else if split.Expense.PaidByUserID == otherUserID && split.UserID == currentUserID {
            // Other ha pagato, io devo
            balance -= split.Amount
        }
    }
    
    // 4. Sottrai/aggiungi settlements
    var settlements []Settlement
    db.Where("property_id = ? AND ((from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?))",
        propertyID, currentUserID, otherUserID, otherUserID, currentUserID).
       Find(&settlements)
    
    for _, settlement := range settlements {
        if settlement.FromUserID == currentUserID {
            // Io ho pagato
            balance += settlement.Amount
        } else {
            // Io ho ricevuto
            balance -= settlement.Amount
        }
    }
    
    return balance, nil
}
```

### Logica Creazione Expense con Split

```go
func CreateExpenseWithSplit(expense *Expense, splitWithUserIDs []uint, db *gorm.DB) error {
    // 1. Verifica split mode
    var settings HouseholdSettings
    if err := db.Where("property_id = ?", expense.PropertyID).First(&settings).Error; err != nil {
        return err
    }
    
    if !settings.SplitMode || len(splitWithUserIDs) == 0 {
        // Split mode off o nessuno da includere = spesa normale
        expense.IsSplit = false
        return db.Create(expense).Error
    }
    
    // 2. Crea spesa
    expense.IsSplit = true
    if err := db.Create(expense).Error; err != nil {
        return err
    }
    
    // 3. Crea splits
    totalPeople := len(splitWithUserIDs) + 1 // +1 per chi ha pagato
    splitAmount := expense.Amount / float64(totalPeople)
    
    // Split per chi ha pagato (già saldato)
    payerSplit := ExpenseSplit{
        ExpenseID: expense.ID,
        UserID:    expense.PaidByUserID,
        Amount:    splitAmount,
        IsSettled: true,
        SettledAt: &expense.Date, // Considerato saldato immediatamente
    }
    if err := db.Create(&payerSplit).Error; err != nil {
        return err
    }
    
    // Split per gli altri (non saldati)
    for _, userID := range splitWithUserIDs {
        split := ExpenseSplit{
            ExpenseID: expense.ID,
            UserID:    userID,
            Amount:    splitAmount,
            IsSettled: false,
        }
        if err := db.Create(&split).Error; err != nil {
            return err
        }
    }
    
    return nil
}
```

### Logica Settlement

```go
func CreateSettlement(settlement *Settlement, db *gorm.DB) error {
    // 1. Validazione
    if settlement.FromUserID == settlement.ToUserID {
        return errors.New("from_user and to_user cannot be the same")
    }
    if settlement.Amount <= 0 {
        return errors.New("amount must be positive")
    }
    
    // 2. Crea settlement
    if err := db.Create(settlement).Error; err != nil {
        return err
    }
    
    // 3. Marca splits come saldati
    // Trova tutti gli split non saldati tra from e to
    var splits []ExpenseSplit
    db.Where("is_settled = ? AND expense_id IN (?)", 
        false,
        db.Table("expenses").
            Select("id").
            Where("property_id = ? AND paid_by_user_id IN (?) AND user_id IN (?)",
                settlement.PropertyID,
                []uint{settlement.FromUserID, settlement.ToUserID},
                []uint{settlement.FromUserID, settlement.ToUserID})).
       Find(&splits)
    
    remainingAmount := settlement.Amount
    now := time.Now()
    
    for i := range splits {
        if remainingAmount <= 0 {
            break
        }
        
        if splits[i].Amount <= remainingAmount {
            // Salda completamente questo split
            splits[i].IsSettled = true
            splits[i].SettledAt = &now
            splits[i].SettlementID = &settlement.ID
            remainingAmount -= splits[i].Amount
            db.Save(&splits[i])
        }
    }
    
    return nil
}
```

---

## 🔌 API ENDPOINTS

### 1. GET `/api/v1/properties/:propertyId/balance`
Ottieni bilancio corrente per una property

**Query Params:**
- `other_user_id` (optional): ID dell'altro utente (se non fornito, usa il partner della property)

**Response:**
```json
{
  "balance": 123.45,
  "current_user_id": 1,
  "other_user_id": 2,
  "current_user_name": "Simone",
  "other_user_name": "Valentina",
  "currency": "EUR",
  "unsettled_expenses_count": 12,
  "total_you_paid": 450.00,
  "total_other_paid": 326.55,
  "message": "Valentina ti deve €123.45"
}
```

### 2. GET `/api/v1/properties/:propertyId/balance/details`
Dettagli bilancio (split non saldati + settlements)

**Response:**
```json
{
  "balance": 123.45,
  "unsettled_splits": [
    {
      "expense_id": 5,
      "expense_description": "Spesa Conad",
      "expense_date": "2025-01-27",
      "expense_category": "Alimentari",
      "paid_by_user_id": 1,
      "paid_by_name": "Simone",
      "split_amount": 60.00,
      "my_share": true
    }
  ],
  "settlements": [
    {
      "id": 3,
      "date": "2025-01-20",
      "from_user_name": "Valentina",
      "to_user_name": "Simone",
      "amount": 150.00,
      "payment_method": "bank_transfer",
      "note": "Pareggio gennaio"
    }
  ]
}
```

### 3. POST `/api/v1/settlements`
Registra un nuovo settlement (pagamento)

**Request:**
```json
{
  "property_id": 1,
  "from_user_id": 2,
  "to_user_id": 1,
  "amount": 123.45,
  "date": "2025-01-27",
  "payment_method": "bank_transfer",
  "note": "Pareggio gennaio 2025"
}
```

**Response:**
```json
{
  "id": 5,
  "property_id": 1,
  "from_user_id": 2,
  "to_user_id": 1,
  "amount": 123.45,
  "date": "2025-01-27",
  "payment_method": "bank_transfer",
  "note": "Pareggio gennaio 2025",
  "created_at": "2025-01-27T14:30:00Z",
  "splits_settled": 8
}
```

### 4. GET `/api/v1/settlements`
Lista settlements di una property

**Query Params:**
- `property_id` (required)
- `from_date` (optional)
- `to_date` (optional)
- `limit` (optional, default: 50)

**Response:**
```json
{
  "settlements": [
    {
      "id": 5,
      "date": "2025-01-27",
      "from_user_name": "Valentina",
      "to_user_name": "Simone",
      "amount": 123.45,
      "payment_method": "bank_transfer",
      "note": "Pareggio gennaio"
    }
  ],
  "total": 15,
  "page": 1
}
```

### 5. POST `/api/v1/expenses` (MODIFICATO)
Crea spesa con supporto split

**Request:**
```json
{
  "amount": 120.00,
  "description": "Spesa Conad",
  "category_id": 1,
  "property_id": 1,
  "date": "2025-01-27",
  "paid_by_user_id": 1,
  "is_split": true,
  "split_with_user_ids": [2]
}
```

**Response:**
```json
{
  "id": 10,
  "amount": 120.00,
  "description": "Spesa Conad",
  "paid_by_user_id": 1,
  "is_split": true,
  "splits": [
    {
      "user_id": 1,
      "amount": 60.00,
      "is_settled": true
    },
    {
      "user_id": 2,
      "amount": 60.00,
      "is_settled": false
    }
  ],
  "created_at": "2025-01-27T14:30:00Z"
}
```

### 6. PUT `/api/v1/properties/:propertyId/settings`
Aggiorna impostazioni household (split mode on/off)

**Request:**
```json
{
  "split_mode": true,
  "default_split_type": "equal"
}
```

**Response:**
```json
{
  "id": 1,
  "property_id": 1,
  "split_mode": true,
  "default_split_type": "equal",
  "updated_at": "2025-01-27T14:30:00Z"
}
```

---

## 🎨 FRONTEND COMPONENTS

### 1. BalanceCard (Dashboard)
```jsx
const BalanceCard = ({ balance, otherUser, onSettleClick }) => {
  return (
    <Card className="backdrop-blur-xl bg-opacity-70 border-2 border-blue-500">
      <h3 className="text-xl font-bold">Bilancio con {otherUser.name}</h3>
      
      <div className="grid grid-cols-3 gap-4 my-4">
        {/* Bilancio Netto */}
        <div className={balance > 0 ? 'bg-green-50' : 'bg-red-50'}>
          <div className="text-sm text-gray-600">Bilancio Netto</div>
          <div className={`text-3xl font-bold ${balance > 0 ? 'text-green-600' : 'text-red-600'}`}>
            {balance > 0 ? '+' : ''}{balance.toFixed(2)} €
          </div>
        </div>
        
        {/* Spese da te pagate */}
        <StatCard label="Spese da te pagate" amount={yourTotal} />
        
        {/* Spese pagate da partner */}
        <StatCard label={`Spese pagate da ${otherUser.name}`} amount={theirTotal} />
      </div>
      
      <div className="flex gap-3">
        <Button 
          onClick={onSettleClick}
          variant={balance !== 0 ? 'success' : 'secondary'}
        >
          {balance > 0 ? 'Ricevi pagamento' : balance < 0 ? 'Salda conto' : 'Pareggio'}
        </Button>
        <Button variant="secondary" onClick={() => navigate('/settlements')}>
          Storico pagamenti
        </Button>
      </div>
      
      {/* Preview spese non saldate */}
      <UnsetledExpensesPreview expenses={unsettledExpenses.slice(0, 3)} />
    </Card>
  );
};
```

### 2. SettlementModal
```jsx
const SettlementModal = ({ balance, otherUser, onConfirm, onCancel }) => {
  const [amount, setAmount] = useState(Math.abs(balance));
  const [date, setDate] = useState(new Date().toISOString().split('T')[0]);
  const [paymentMethod, setPaymentMethod] = useState('bank_transfer');
  const [note, setNote] = useState('');
  
  return (
    <Modal>
      <h3>{balance > 0 ? 'Ricevi Pagamento' : 'Salda Conto'}</h3>
      
      <div className="bg-blue-50 p-4 rounded-lg text-center">
        <div className="text-sm text-gray-600">Importo da saldare</div>
        <div className="text-4xl font-bold text-blue-600">
          {Math.abs(balance).toFixed(2)} €
        </div>
        <div className="text-sm mt-2">
          {balance > 0 ? `Da ricevere da ${otherUser.name}` : `Da pagare a ${otherUser.name}`}
        </div>
      </div>
      
      <Input 
        label="Importo effettivo" 
        type="number" 
        value={amount}
        onChange={(e) => setAmount(e.target.value)}
      />
      
      <Input 
        label="Data pagamento" 
        type="date" 
        value={date}
        onChange={(e) => setDate(e.target.value)}
      />
      
      <Select 
        label="Metodo pagamento"
        value={paymentMethod}
        onChange={(e) => setPaymentMethod(e.target.value)}
      >
        <option value="bank_transfer">Bonifico bancario</option>
        <option value="cash">Contanti</option>
        <option value="satispay">Satispay</option>
        <option value="paypal">PayPal</option>
      </Select>
      
      <Textarea 
        label="Nota (opzionale)" 
        value={note}
        onChange={(e) => setNote(e.target.value)}
        placeholder="Es: Pareggio gennaio 2025"
      />
      
      <Alert variant="warning">
        Registrando questo pagamento, le spese collegate verranno marcate come saldate
      </Alert>
      
      <div className="flex gap-3">
        <Button variant="secondary" onClick={onCancel}>Annulla</Button>
        <Button variant="success" onClick={() => onConfirm({ amount, date, paymentMethod, note })}>
          Conferma Pagamento
        </Button>
      </div>
    </Modal>
  );
};
```

### 3. AddExpenseModal (con Split)
```jsx
const AddExpenseModal = ({ familyMembers, splitModeEnabled, currentUser, onSave }) => {
  const [paidBy, setPaidBy] = useState(currentUser.id);
  const [splitWith, setSplitWith] = useState([]);
  const [customSplit, setCustomSplit] = useState(false);
  
  return (
    <Modal>
      <h3>Nuova Spesa</h3>
      
      {/* Campi standard: importo, descrizione, categoria, data */}
      <Input label="Importo" type="number" />
      <Input label="Descrizione" type="text" />
      <Select label="Categoria">...</Select>
      <Input label="Data" type="date" />
      
      {/* SEZIONE SPLIT */}
      {splitModeEnabled && (
        <div className="bg-blue-50 p-4 rounded-lg space-y-3">
          <h4 className="font-medium">Divisione Spesa</h4>
          
          {/* Chi ha pagato */}
          <div>
            <label>Pagato da:</label>
            {familyMembers.map(member => (
              <Radio 
                key={member.id}
                checked={paidBy === member.id}
                onChange={() => setPaidBy(member.id)}
                label={member.name}
              />
            ))}
          </div>
          
          {/* Dividi con */}
          <div>
            <label>Dividi con:</label>
            {familyMembers.filter(m => m.id !== paidBy).map(member => (
              <Checkbox
                key={member.id}
                checked={splitWith.includes(member.id)}
                onChange={(checked) => {
                  if (checked) {
                    setSplitWith([...splitWith, member.id]);
                  } else {
                    setSplitWith(splitWith.filter(id => id !== member.id));
                  }
                }}
                label={member.name}
              />
            ))}
          </div>
          
          {/* Divisione personalizzata */}
          <Checkbox 
            checked={customSplit}
            onChange={setCustomSplit}
            label="Divisione personalizzata"
          />
          
          {!customSplit && splitWith.length > 0 && (
            <div className="text-sm text-gray-600 p-2 bg-white rounded">
              💡 La spesa sarà divisa equamente tra {splitWith.length + 1} persone
            </div>
          )}
        </div>
      )}
      
      <div className="flex gap-3">
        <Button variant="secondary">Annulla</Button>
        <Button onClick={() => onSave({ paidBy, splitWith })}>Salva</Button>
      </div>
    </Modal>
  );
};
```

### 4. SettlementsView
```jsx
const SettlementsView = () => {
  const { balance, settlements } = useSettlements();
  
  return (
    <div>
      <Header>
        <h2>Storico Pagamenti</h2>
        <Button onClick={openSettlementModal}>Nuovo Pagamento</Button>
      </Header>
      
      {/* Balance Summary */}
      <Card>
        <BalanceSummary balance={balance} />
      </Card>
      
      {/* Settlements List */}
      <Card>
        <h3>Pagamenti Effettuati</h3>
        {settlements.map(settlement => (
          <SettlementItem 
            key={settlement.id}
            settlement={settlement}
            currentUserId={currentUser.id}
          />
        ))}
      </Card>
    </div>
  );
};
```

### 5. SettingsView (con toggle Split Mode)
```jsx
const SettingsView = () => {
  const [splitMode, setSplitMode] = useState(false);
  
  return (
    <div>
      <Card>
        <h3>Gestione Divisione Spese</h3>
        
        {/* Toggle Split Mode */}
        <div className="flex items-center justify-between">
          <div>
            <div className="font-medium">Modalità Divisione Spese</div>
            <div className="text-sm text-gray-600">
              Attiva per tenere traccia di chi deve cosa a chi
            </div>
          </div>
          <Toggle 
            checked={splitMode}
            onChange={async (checked) => {
              await updateHouseholdSettings({ split_mode: checked });
              setSplitMode(checked);
            }}
          />
        </div>
        
        {splitMode && (
          <>
            {/* Membri famiglia */}
            <FamilyMembers members={familyMembers} />
            
            {/* Regole divisione */}
            <Select label="Regole Divisione" value={splitType} onChange={setSplitType}>
              <option value="equal">50/50 - Divisione equa</option>
              <option value="custom">Personalizzata per spesa</option>
              <option value="income_based">In base al reddito</option>
            </Select>
            
            {/* Warning se bilancio != 0 */}
            {balance !== 0 && (
              <Alert variant="warning">
                Bilancio Attuale: {balance > 0 ? `+${balance}` : balance} €
              </Alert>
            )}
          </>
        )}
      </Card>
    </div>
  );
};
```

---

## 🔄 USER FLOW

### Flow 1: Attivazione Split Mode
1. Utente va in Impostazioni
2. Attiva toggle "Modalità Divisione Spese"
3. Sistema crea HouseholdSettings per la property
4. Badge "Split Mode ON" appare in navbar
5. Tutte le nuove spese avranno opzione split

### Flow 2: Aggiungi Spesa con Split
1. Utente clicca FAB "+"
2. Compila importo, descrizione, categoria, data
3. **NUOVO**: Seleziona "Pagato da" (default: utente corrente)
4. **NUOVO**: Seleziona "Dividi con" (checkbox membri famiglia)
5. Sistema calcola split automaticamente (es: €120 / 2 = €60 ciascuno)
6. Salva expense + crea ExpenseSplit per ogni membro
7. Bilancio si aggiorna automaticamente in dashboard

### Flow 3: Visualizza Bilancio
1. Dashboard mostra card "Bilancio con [Nome]"
2. Visualizza:
   - Bilancio netto (+€123.45 o -€123.45)
   - Totale spese pagate da te
   - Totale spese pagate da partner
3. Preview 3 spese non saldate
4. Badge in navbar se bilancio ≠ 0

### Flow 4: Salda Conto
1. Utente clicca "Salda conto" da dashboard o pagamenti
2. Modale mostra importo da saldare
3. Utente conferma importo, data, metodo, nota
4. Sistema:
   - Crea Settlement
   - Marca ExpenseSplit come saldati
   - Aggiorna bilancio a 0 (o riduce)
5. Conferma "Pagamento registrato!"
6. Badge sparisce se bilancio = 0

### Flow 5: Storico Pagamenti
1. Utente naviga a sezione "Pagamenti"
2. Visualizza:
   - Bilancio corrente (grande)
   - Lista settlements con dettagli
3. Filtri: data, metodo pagamento
4. Pulsante "Nuovo Pagamento" sempre visibile

---

## 🧪 TEST CASES

### Test 1: Split Equo Semplice
```
Given: Split mode attivo, 2 membri (Simone, Valentina)
When: Simone aggiunge spesa €100, split con Valentina
Then:
  - Expense.paid_by_user_id = 1 (Simone)
  - ExpenseSplit.user_id = 1, amount = 50, is_settled = true
  - ExpenseSplit.user_id = 2, amount = 50, is_settled = false
  - Balance Simone = +€50
  - Balance Valentina = -€50
```

### Test 2: Multiple Spese
```
Given: 
  - Simone paga €100 (split)
  - Valentina paga €60 (split)
  - Simone paga €80 (split)
When: Calcolo bilancio
Then:
  - Simone: +€50 (da spesa 1) -€30 (da spesa 2) +€40 (da spesa 3) = +€60
  - Valentina: -€50 +€30 -€40 = -€60
```

### Test 3: Settlement Completo
```
Given: Valentina deve a Simone €60
When: Valentina registra settlement di €60
Then:
  - Balance Simone = 0
  - Balance Valentina = 0
  - Tutti gli ExpenseSplit marcati is_settled = true
```

### Test 4: Settlement Parziale
```
Given: Valentina deve a Simone €100
When: Valentina registra settlement di €50
Then:
  - Balance Simone = +€50
  - Balance Valentina = -€50
  - Solo gli split più vecchi per totale €50 marcati is_settled = true
```

### Test 5: Split Mode Disattivato
```
Given: Split mode OFF
When: Utente aggiunge spesa
Then:
  - Nessun ExpenseSplit creato
  - Balance sempre 0
  - Opzioni split non visibili in UI
```

### Test 6: Cambio Pagatore
```
Given: Split mode attivo
When: Utente aggiunge spesa e seleziona "Pagato da Valentina"
Then:
  - Expense.paid_by_user_id = 2 (Valentina)
  - Split per Valentina is_settled = true
  - Split per Simone is_settled = false
  - Balance aggiornato correttamente
```

---

## 🚀 IMPLEMENTATION STEPS

### Phase 1: Database (1 giorno)
1. ✅ Creare migration per nuovi modelli
2. ✅ Aggiornare modello Expense
3. ✅ Testare migrations su DB test

### Phase 2: Backend Logic (2 giorni)
1. ✅ Implementare `CalculateBalance()`
2. ✅ Implementare `CreateExpenseWithSplit()`
3. ✅ Implementare `CreateSettlement()`
4. ✅ Scrivere unit tests per logica
5. ✅ Validazioni e error handling

### Phase 3: Backend API (1 giorno)
1. ✅ Endpoint GET balance
2. ✅ Endpoint GET balance/details
3. ✅ Endpoint POST settlements
4. ✅ Endpoint GET settlements
5. ✅ Aggiornare POST expenses
6. ✅ Endpoint PUT household settings

### Phase 4: Frontend Components (2 giorni)
1. ✅ BalanceCard per dashboard
2. ✅ AddExpenseModal con split
3. ✅ SettlementModal
4. ✅ SettlementsView
5. ✅ Toggle split mode in settings
6. ✅ Badge notifica in navbar

### Phase 5: Frontend Integration (1 giorno)
1. ✅ Hook useBalance()
2. ✅ Hook useSettlements()
3. ✅ Integrazione API calls
4. ✅ State management (Pinia)
5. ✅ Real-time updates

### Phase 6: Testing (1 giorno)
1. ✅ Unit tests frontend
2. ✅ Integration tests API
3. ✅ E2E tests user flow
4. ✅ Performance tests (1000+ spese)

### Phase 7: Polish (0.5 giorni)
1. ✅ Loading states
2. ✅ Error messages
3. ✅ Animations
4. ✅ Mobile responsive check

**Totale stimato: 8.5 giorni**

---

## 📊 ANALYTICS & INSIGHTS

### Metriche da Tracciare
1. **Bilancio medio** mensile
2. **Numero settlements** per mese
3. **Tempo medio** tra spesa e settlement
4. **Percentuale spese split** vs non-split
5. **Chi paga di più** (distribuzione)

### Dashboard Analytics
```
📊 Questo Mese:
- Spese totali: €1,245
- Tua quota: €622.50 (50%)
- Spese da te pagate: €670 (54%)
- Spese pagate da partner: €575 (46%)
- Settlements: 2 (totale €150)
- Bilancio corrente: +€47.50
```

---

## 🔒 SECURITY & PRIVACY

### Validazioni
1. Solo membri della stessa property possono vedere bilanci
2. Solo admin può attivare/disattivare split mode
3. Settlement amount deve essere ≤ balance
4. Non si possono creare settlements tra utenti di property diverse

### Privacy
1. Bilanci visibili solo ai membri coinvolti
2. Settlements non condivisi pubblicamente
3. Soft delete per mantenere storico
4. Export dati GDPR compliant

---

## 📝 NOTES & ASSUMPTIONS

### Assumptions
1. Massimo 2 membri per property (MVP)
2. Split sempre 50/50 (MVP)
3. Una property = una coppia/famiglia
4. Settlements manuali (no integrazione bancaria)

### Future Enhancements
1. ⏭️ Split personalizzato per spesa (60/40, etc.)
2. ⏭️ Supporto 3+ membri
3. ⏭️ Recurring settlements automatici
4. ⏭️ Integrazione PSD2 banche
5. ⏭️ Notifiche push "Bilancio > €X"
6. ⏭️ Export PDF report mensile

---

**Versione:** 1.0  
**Data:** 27 Gennaio 2026  
**Status:** Ready for Implementation 🚀