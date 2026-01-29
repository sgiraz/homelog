import React, { useState } from 'react';
import { Home, Receipt, TrendingUp, Droplet, Settings, Plus, Users, ArrowLeftRight, DollarSign, CheckCircle, AlertCircle, ChevronRight, X } from 'lucide-react';

// Mock Data con split
const mockUsers = [
  { id: 1, name: 'Simone', email: 's.girardi92@gmail.com', role: 'admin' },
  { id: 2, name: 'Valentina', email: 'vale@example.com', role: 'user' },
];

const mockExpensesWithSplit = [
  { 
    id: 1, 
    date: '2025-01-27', 
    category: '🍕 Alimentari', 
    amount: 45.00, 
    description: 'Pizzeria Da Mario',
    paidBy: 1, // Simone
    splits: [
      { userId: 1, amount: 22.50, settled: true },
      { userId: 2, amount: 22.50, settled: false }
    ]
  },
  { 
    id: 2, 
    date: '2025-01-26', 
    category: '🏠 Casa', 
    amount: 58.00, 
    description: 'Bolletta Luce E.ON',
    paidBy: 2, // Valentina
    splits: [
      { userId: 1, amount: 29.00, settled: false },
      { userId: 2, amount: 29.00, settled: true }
    ]
  },
  { 
    id: 3, 
    date: '2025-01-25', 
    category: '🏠 Casa', 
    amount: 120.00, 
    description: 'Spesa Conad',
    paidBy: 1, // Simone
    splits: [
      { userId: 1, amount: 60.00, settled: true },
      { userId: 2, amount: 60.00, settled: false }
    ]
  },
];

const mockSettlements = [
  { id: 1, date: '2025-01-20', fromUser: 2, toUser: 1, amount: 150.00, note: 'Pareggio gennaio' },
  { id: 2, date: '2025-01-10', fromUser: 1, toUser: 2, amount: 80.00, note: 'Spese natalizie' },
];

const HomeLogSplitPrototype = () => {
  const [currentView, setCurrentView] = useState('dashboard');
  const [darkMode, setDarkMode] = useState(false);
  const [splitModeEnabled, setSplitModeEnabled] = useState(true);
  const [currentUser] = useState(1); // Simone
  const [showAddExpense, setShowAddExpense] = useState(false);
  const [showSettlement, setShowSettlement] = useState(false);

  // Calcola bilancio
  const calculateBalance = () => {
    let balance = 0;
    mockExpensesWithSplit.forEach(expense => {
      expense.splits.forEach(split => {
        if (!split.settled) {
          if (expense.paidBy === currentUser && split.userId !== currentUser) {
            // Qualcuno mi deve
            balance += split.amount;
          } else if (expense.paidBy !== currentUser && split.userId === currentUser) {
            // Devo a qualcuno
            balance -= split.amount;
          }
        }
      });
    });
    return balance;
  };

  const balance = calculateBalance();
  const otherUser = mockUsers.find(u => u.id !== currentUser);

  const bgColor = darkMode ? 'bg-gray-900' : 'bg-gray-50';
  const cardBg = darkMode ? 'bg-gray-800' : 'bg-white';
  const textColor = darkMode ? 'text-gray-100' : 'text-gray-900';
  const textSecondary = darkMode ? 'text-gray-400' : 'text-gray-600';
  const borderColor = darkMode ? 'border-gray-700' : 'border-gray-200';

  const Card = ({ children, className = '' }) => (
    <div className={`${cardBg} rounded-xl border ${borderColor} ${className}`}>
      {children}
    </div>
  );

  const Button = ({ children, onClick, variant = 'primary', className = '' }) => {
    const variants = {
      primary: 'bg-blue-600 hover:bg-blue-700 text-white',
      secondary: 'bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-900 dark:text-white',
      success: 'bg-green-600 hover:bg-green-700 text-white',
      danger: 'bg-red-600 hover:bg-red-700 text-white',
    };
    
    return (
      <button
        onClick={onClick}
        className={`px-4 py-2 rounded-lg font-medium transition-colors ${variants[variant]} ${className}`}
      >
        {children}
      </button>
    );
  };

  // Dashboard View with Balance Section
  const DashboardView = () => (
    <div className="space-y-6">
      {/* Balance Section - NUOVA */}
      {splitModeEnabled && (
        <Card className="p-6 backdrop-blur-xl bg-opacity-70 border-2 border-blue-500">
          <div className="flex items-center justify-between mb-4">
            <h3 className={`text-xl font-bold ${textColor} flex items-center gap-2`}>
              <ArrowLeftRight className="text-blue-500" size={24} />
              Bilancio con {otherUser.name}
            </h3>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
            {/* Bilancio Netto */}
            <div className={`p-4 rounded-xl ${balance > 0 ? 'bg-green-50 dark:bg-green-900/20' : balance < 0 ? 'bg-red-50 dark:bg-red-900/20' : 'bg-gray-50 dark:bg-gray-700'}`}>
              <div className={`text-sm ${textSecondary} mb-1`}>Bilancio Netto</div>
              <div className={`text-3xl font-bold ${balance > 0 ? 'text-green-600' : balance < 0 ? 'text-red-600' : textColor}`}>
                {balance > 0 ? '+' : ''}{balance.toFixed(2)} €
              </div>
              <div className="text-xs mt-1">
                {balance > 0 ? `${otherUser.name} ti deve` : balance < 0 ? `Tu devi a ${otherUser.name}` : 'Siete in pari'}
              </div>
            </div>

            {/* Spese da te pagate */}
            <div className={`p-4 rounded-xl bg-blue-50 dark:bg-blue-900/20`}>
              <div className={`text-sm ${textSecondary} mb-1`}>Spese da te pagate</div>
              <div className={`text-2xl font-bold text-blue-600`}>
                {mockExpensesWithSplit.filter(e => e.paidBy === currentUser).reduce((sum, e) => sum + e.amount, 0).toFixed(2)} €
              </div>
              <div className="text-xs mt-1">
                {mockExpensesWithSplit.filter(e => e.paidBy === currentUser).length} spese
              </div>
            </div>

            {/* Spese pagate da partner */}
            <div className={`p-4 rounded-xl bg-purple-50 dark:bg-purple-900/20`}>
              <div className={`text-sm ${textSecondary} mb-1`}>Spese pagate da {otherUser.name}</div>
              <div className={`text-2xl font-bold text-purple-600`}>
                {mockExpensesWithSplit.filter(e => e.paidBy !== currentUser).reduce((sum, e) => sum + e.amount, 0).toFixed(2)} €
              </div>
              <div className="text-xs mt-1">
                {mockExpensesWithSplit.filter(e => e.paidBy !== currentUser).length} spese
              </div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex gap-3">
            <Button 
              onClick={() => setShowSettlement(true)} 
              variant={balance !== 0 ? 'success' : 'secondary'}
              className="flex-1 flex items-center justify-center gap-2"
            >
              <CheckCircle size={20} />
              {balance > 0 ? 'Ricevi pagamento' : balance < 0 ? 'Salda conto' : 'Pareggio'}
            </Button>
            <Button 
              onClick={() => setCurrentView('settlements')}
              variant="secondary"
              className="flex-1 flex items-center justify-center gap-2"
            >
              <Receipt size={20} />
              Storico pagamenti
            </Button>
          </div>

          {/* Dettaglio spese non saldate */}
          {balance !== 0 && (
            <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
              <div className="text-sm font-medium mb-2">Spese da saldare:</div>
              <div className="space-y-2">
                {mockExpensesWithSplit.filter(expense => 
                  expense.splits.some(s => !s.settled && ((expense.paidBy === currentUser && s.userId !== currentUser) || (expense.paidBy !== currentUser && s.userId === currentUser)))
                ).slice(0, 3).map(expense => {
                  const split = expense.splits.find(s => s.userId === (expense.paidBy === currentUser ? otherUser.id : currentUser));
                  return (
                    <div key={expense.id} className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-700 rounded-lg text-sm">
                      <div className="flex-1">
                        <div className={textColor}>{expense.description}</div>
                        <div className={`text-xs ${textSecondary}`}>{expense.date} • {expense.category}</div>
                      </div>
                      <div className={`font-bold ${expense.paidBy === currentUser ? 'text-green-600' : 'text-red-600'}`}>
                        {expense.paidBy === currentUser ? '+' : '-'}{split.amount.toFixed(2)} €
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </Card>
      )}

      {/* KPI Cards originali */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="p-6">
          <div className="flex items-center justify-between mb-2">
            <span className={`${textSecondary} text-sm font-medium`}>Spese Totali Mese</span>
            <DollarSign className="text-blue-500" size={20} />
          </div>
          <div className={`text-3xl font-bold ${textColor}`}>
            {mockExpensesWithSplit.reduce((sum, e) => sum + e.amount, 0).toFixed(2)} €
          </div>
          <div className="text-sm text-green-500 mt-1">Famiglia</div>
        </Card>

        <Card className="p-6">
          <div className="flex items-center justify-between mb-2">
            <span className={`${textSecondary} text-sm font-medium`}>La tua quota</span>
            <Users className="text-purple-500" size={20} />
          </div>
          <div className={`text-3xl font-bold ${textColor}`}>
            {(mockExpensesWithSplit.reduce((sum, e) => sum + e.amount, 0) / 2).toFixed(2)} €
          </div>
          <div className="text-sm text-gray-500 mt-1">50% del totale</div>
        </Card>

        <Card className="p-6">
          <div className="flex items-center justify-between mb-2">
            <span className={`${textSecondary} text-sm font-medium`}>Spese attive</span>
            <Receipt className="text-orange-500" size={20} />
          </div>
          <div className={`text-3xl font-bold ${textColor}`}>{mockExpensesWithSplit.length}</div>
          <div className="text-sm text-gray-500 mt-1">Questo mese</div>
        </Card>
      </div>

      {/* Recent Expenses with Split Info */}
      <Card className="p-6">
        <h3 className={`text-lg font-semibold ${textColor} mb-4`}>📝 Spese Recenti</h3>
        <div className="space-y-3">
          {mockExpensesWithSplit.map(expense => {
            const payer = mockUsers.find(u => u.id === expense.paidBy);
            const myShare = expense.splits.find(s => s.userId === currentUser);
            
            return (
              <div key={expense.id} className="p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <span className={`text-sm ${textSecondary}`}>{expense.date}</span>
                      <span className="text-sm font-medium">{expense.category}</span>
                    </div>
                    <div className={`font-medium ${textColor} mb-2`}>{expense.description}</div>
                    
                    {splitModeEnabled && (
                      <div className="flex items-center gap-3 text-xs">
                        <span className={textSecondary}>
                          Pagato da: <strong>{payer.name}</strong>
                        </span>
                        <span className={`px-2 py-1 rounded-full ${myShare.settled ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400' : 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-400'}`}>
                          {myShare.settled ? '✓ Saldato' : '⏳ Da saldare'}
                        </span>
                      </div>
                    )}
                  </div>
                  
                  <div className="text-right">
                    <div className={`text-xl font-bold ${textColor}`}>{expense.amount.toFixed(2)} €</div>
                    {splitModeEnabled && (
                      <div className={`text-sm ${textSecondary} mt-1`}>
                        Tua quota: <strong>{myShare.amount.toFixed(2)} €</strong>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </Card>
    </div>
  );

  // Settlements View
  const SettlementsView = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className={`text-2xl font-bold ${textColor}`}>Storico Pagamenti</h2>
        <Button onClick={() => setShowSettlement(true)} className="flex items-center gap-2">
          <Plus size={20} />
          Nuovo Pagamento
        </Button>
      </div>

      {/* Balance Summary */}
      <Card className="p-6">
        <div className="text-center">
          <div className={`text-sm ${textSecondary} mb-2`}>Bilancio Attuale</div>
          <div className={`text-5xl font-bold mb-2 ${balance > 0 ? 'text-green-600' : balance < 0 ? 'text-red-600' : textColor}`}>
            {balance > 0 ? '+' : ''}{balance.toFixed(2)} €
          </div>
          <div className="text-sm">
            {balance > 0 ? `${otherUser.name} ti deve questo importo` : balance < 0 ? `Tu devi a ${otherUser.name} questo importo` : 'Siete in pari! 🎉'}
          </div>
        </div>
      </Card>

      {/* Settlements List */}
      <Card className="p-6">
        <h3 className={`text-lg font-semibold ${textColor} mb-4`}>💸 Pagamenti Effettuati</h3>
        <div className="space-y-3">
          {mockSettlements.map(settlement => {
            const fromUser = mockUsers.find(u => u.id === settlement.fromUser);
            const toUser = mockUsers.find(u => u.id === settlement.toUser);
            const isReceived = settlement.toUser === currentUser;
            
            return (
              <div key={settlement.id} className={`p-4 border ${borderColor} rounded-lg ${isReceived ? 'bg-green-50 dark:bg-green-900/20' : 'bg-blue-50 dark:bg-blue-900/20'}`}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className={`p-2 rounded-full ${isReceived ? 'bg-green-100 dark:bg-green-900/50' : 'bg-blue-100 dark:bg-blue-900/50'}`}>
                      <ArrowLeftRight className={isReceived ? 'text-green-600' : 'text-blue-600'} size={20} />
                    </div>
                    <div>
                      <div className={`font-medium ${textColor}`}>
                        {isReceived ? `Ricevuto da ${fromUser.name}` : `Pagato a ${toUser.name}`}
                      </div>
                      <div className={`text-sm ${textSecondary}`}>{settlement.date}</div>
                      {settlement.note && (
                        <div className={`text-xs ${textSecondary} mt-1`}>{settlement.note}</div>
                      )}
                    </div>
                  </div>
                  <div className={`text-2xl font-bold ${isReceived ? 'text-green-600' : 'text-blue-600'}`}>
                    {isReceived ? '+' : '-'}{settlement.amount.toFixed(2)} €
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </Card>
    </div>
  );

  // Settings View with Split Mode Toggle
  const SettingsView = () => (
    <div className="space-y-6">
      <h2 className={`text-2xl font-bold ${textColor}`}>Impostazioni</h2>

      {/* Split Mode Settings - NUOVA SEZIONE */}
      <Card className="p-6">
        <h3 className={`text-lg font-semibold ${textColor} mb-4`}>💰 Gestione Divisione Spese</h3>
        
        <div className="space-y-4">
          <div className="flex items-center justify-between p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
            <div>
              <div className={`font-medium ${textColor}`}>Modalità Divisione Spese</div>
              <div className={`text-sm ${textSecondary} mt-1`}>
                Attiva per tenere traccia di chi deve cosa a chi
              </div>
            </div>
            <button
              onClick={() => setSplitModeEnabled(!splitModeEnabled)}
              className={`relative w-14 h-8 rounded-full transition-colors ${splitModeEnabled ? 'bg-green-600' : 'bg-gray-300'}`}
            >
              <div className={`absolute top-1 left-1 w-6 h-6 bg-white rounded-full transition-transform ${splitModeEnabled ? 'translate-x-6' : ''}`} />
            </button>
          </div>

          {splitModeEnabled && (
            <>
              <div className="p-4 border border-gray-200 dark:border-gray-700 rounded-lg">
                <div className={`font-medium ${textColor} mb-2`}>Membri Famiglia</div>
                <div className="space-y-2">
                  {mockUsers.map(user => (
                    <div key={user.id} className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-700 rounded">
                      <div className="flex items-center gap-3">
                        <Users size={20} className="text-blue-500" />
                        <div>
                          <div className={`font-medium ${textColor}`}>{user.name}</div>
                          <div className={`text-xs ${textSecondary}`}>{user.email}</div>
                        </div>
                      </div>
                      <span className={`text-xs px-2 py-1 rounded-full ${user.role === 'admin' ? 'bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-400' : 'bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-300'}`}>
                        {user.role === 'admin' ? 'Admin' : 'Membro'}
                      </span>
                    </div>
                  ))}
                </div>
              </div>

              <div className="p-4 border border-gray-200 dark:border-gray-700 rounded-lg">
                <div className={`font-medium ${textColor} mb-2`}>Regole Divisione</div>
                <select className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
                  <option>50/50 - Divisione equa</option>
                  <option>Personalizzata per spesa</option>
                  <option>In base al reddito</option>
                </select>
                <div className={`text-xs ${textSecondary} mt-2`}>
                  Definisci come dividere le spese comuni tra i membri
                </div>
              </div>

              <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
                <div className="flex items-start gap-2">
                  <AlertCircle className="text-yellow-600 flex-shrink-0 mt-0.5" size={20} />
                  <div>
                    <div className="font-medium text-yellow-800 dark:text-yellow-400">Bilancio Attuale</div>
                    <div className="text-sm text-yellow-700 dark:text-yellow-500 mt-1">
                      {balance > 0 ? `${otherUser.name} ti deve ${balance.toFixed(2)} €` : balance < 0 ? `Tu devi a ${otherUser.name} ${Math.abs(balance).toFixed(2)} €` : 'Siete in pari'}
                    </div>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      </Card>

      {/* Other Settings */}
      <Card className="p-6">
        <h3 className={`text-lg font-semibold ${textColor} mb-4`}>👤 Profilo</h3>
        <div className="space-y-3">
          <div>
            <label className={`block text-sm ${textSecondary} mb-1`}>Nome</label>
            <input 
              type="text" 
              value="Simone Girardi" 
              className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
            />
          </div>
          <div>
            <label className={`block text-sm ${textSecondary} mb-1`}>Email</label>
            <input 
              type="email" 
              value="s.girardi92@gmail.com" 
              className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
            />
          </div>
        </div>
      </Card>

      <Card className="p-6">
        <h3 className={`text-lg font-semibold ${textColor} mb-4`}>🌍 Localizzazione</h3>
        <div className="space-y-3">
          <div>
            <label className={`block text-sm ${textSecondary} mb-1`}>Valuta</label>
            <select className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
              <option>EUR (€)</option>
              <option>USD ($)</option>
            </select>
          </div>
        </div>
      </Card>
    </div>
  );

  // Add Expense Modal with Split Options
  const AddExpenseModal = () => {
    const [paidBy, setPaidBy] = useState(currentUser);
    const [splitWith, setSplitWith] = useState([otherUser.id]);
    const [customSplit, setCustomSplit] = useState(false);
    
    return (
      <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={() => setShowAddExpense(false)}>
        <div className={`${cardBg} rounded-2xl p-6 max-w-md w-full max-h-[90vh] overflow-y-auto`} onClick={e => e.stopPropagation()}>
          <div className="flex items-center justify-between mb-6">
            <h3 className={`text-xl font-bold ${textColor}`}>Nuova Spesa</h3>
            <button onClick={() => setShowAddExpense(false)}>
              <X className={textColor} />
            </button>
          </div>

          <div className="space-y-4">
            <div>
              <label className={`block text-sm ${textSecondary} mb-1`}>Importo *</label>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">€</span>
                <input 
                  type="number" 
                  className={`w-full pl-8 pr-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
                  placeholder="0,00"
                />
              </div>
            </div>

            <div>
              <label className={`block text-sm ${textSecondary} mb-1`}>Descrizione *</label>
              <input 
                type="text" 
                className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
                placeholder="Es: Spesa supermercato"
              />
            </div>

            <div>
              <label className={`block text-sm ${textSecondary} mb-1`}>Categoria *</label>
              <select className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
                <option>🍕 Alimentari</option>
                <option>🏠 Casa</option>
                <option>🚗 Trasporti</option>
              </select>
            </div>

            {/* NUOVA SEZIONE SPLIT */}
            {splitModeEnabled && (
              <div className="p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg space-y-3">
                <div className="font-medium text-blue-800 dark:text-blue-400 flex items-center gap-2">
                  <Users size={18} />
                  Divisione Spesa
                </div>

                <div>
                  <label className={`block text-sm ${textSecondary} mb-2`}>Pagato da:</label>
                  <div className="space-y-2">
                    {mockUsers.map(user => (
                      <label key={user.id} className="flex items-center gap-2 cursor-pointer">
                        <input
                          type="radio"
                          name="paidBy"
                          checked={paidBy === user.id}
                          onChange={() => setPaidBy(user.id)}
                          className="w-4 h-4"
                        />
                        <span className={textColor}>{user.name} {user.id === currentUser && '(tu)'}</span>
                      </label>
                    ))}
                  </div>
                </div>

                <div>
                  <label className={`block text-sm ${textSecondary} mb-2`}>Dividi con:</label>
                  <div className="space-y-2">
                    {mockUsers.filter(u => u.id !== paidBy).map(user => (
                      <label key={user.id} className="flex items-center gap-2 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={splitWith.includes(user.id)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSplitWith([...splitWith, user.id]);
                            } else {
                              setSplitWith(splitWith.filter(id => id !== user.id));
                            }
                          }}
                          className="w-4 h-4"
                        />
                        <span className={textColor}>{user.name}</span>
                      </label>
                    ))}
                  </div>
                </div>

                <div className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={customSplit}
                    onChange={(e) => setCustomSplit(e.target.checked)}
                    className="w-4 h-4"
                  />
                  <span className={textColor}>Divisione personalizzata</span>
                </div>

                {!customSplit && splitWith.length > 0 && (
                  <div className={`text-sm ${textSecondary} p-2 bg-white dark:bg-gray-800 rounded`}>
                    💡 La spesa sarà divisa equamente tra {splitWith.length + 1} persone
                  </div>
                )}
              </div>
            )}

            <div>
              <label className={`block text-sm ${textSecondary} mb-1`}>Data *</label>
              <input 
                type="date" 
                className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
                defaultValue="2025-01-27"
              />
            </div>
          </div>

          <div className="flex gap-3 mt-6">
            <Button variant="secondary" onClick={() => setShowAddExpense(false)} className="flex-1">
              Annulla
            </Button>
            <Button onClick={() => setShowAddExpense(false)} className="flex-1">
              Salva
            </Button>
          </div>
        </div>
      </div>
    );
  };

  // Settlement Modal
  const SettlementModal = () => (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={() => setShowSettlement(false)}>
      <div className={`${cardBg} rounded-2xl p-6 max-w-md w-full`} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-6">
          <h3 className={`text-xl font-bold ${textColor}`}>
            {balance > 0 ? 'Ricevi Pagamento' : 'Salda Conto'}
          </h3>
          <button onClick={() => setShowSettlement(false)}>
            <X className={textColor} />
          </button>
        </div>

        <div className="space-y-4">
          <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg text-center">
            <div className={`text-sm ${textSecondary} mb-2`}>Importo da saldare</div>
            <div className={`text-4xl font-bold ${balance > 0 ? 'text-green-600' : 'text-red-600'}`}>
              {Math.abs(balance).toFixed(2)} €
            </div>
            <div className="text-sm mt-2">
              {balance > 0 ? `Da ricevere da ${otherUser.name}` : `Da pagare a ${otherUser.name}`}
            </div>
          </div>

          <div>
            <label className={`block text-sm ${textSecondary} mb-1`}>Importo effettivo</label>
            <div className="relative">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">€</span>
              <input 
                type="number" 
                defaultValue={Math.abs(balance).toFixed(2)}
                className={`w-full pl-8 pr-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
              />
            </div>
          </div>

          <div>
            <label className={`block text-sm ${textSecondary} mb-1`}>Data pagamento</label>
            <input 
              type="date" 
              defaultValue="2025-01-27"
              className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
            />
          </div>

          <div>
            <label className={`block text-sm ${textSecondary} mb-1`}>Nota (opzionale)</label>
            <textarea 
              className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
              rows="2"
              placeholder="Es: Pareggio gennaio 2025"
            />
          </div>

          <div>
            <label className={`block text-sm ${textSecondary} mb-1`}>Metodo pagamento</label>
            <select className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
              <option>Bonifico bancario</option>
              <option>Contanti</option>
              <option>Satispay</option>
              <option>PayPal</option>
            </select>
          </div>

          <div className="p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg text-sm">
            <div className="flex items-start gap-2">
              <AlertCircle className="text-yellow-600 flex-shrink-0 mt-0.5" size={16} />
              <div className="text-yellow-800 dark:text-yellow-400">
                Registrando questo pagamento, le spese collegate verranno marcate come saldate
              </div>
            </div>
          </div>
        </div>

        <div className="flex gap-3 mt-6">
          <Button variant="secondary" onClick={() => setShowSettlement(false)} className="flex-1">
            Annulla
          </Button>
          <Button variant="success" onClick={() => setShowSettlement(false)} className="flex-1">
            Conferma Pagamento
          </Button>
        </div>
      </div>
    </div>
  );

  const Navbar = () => (
    <nav className={`${darkMode ? 'bg-gray-800 border-gray-700' : 'bg-white border-gray-200'} border-b px-6 py-4 sticky top-0 z-50`}>
      <div className="flex items-center justify-between max-w-7xl mx-auto">
        <div className="flex items-center gap-4">
          <h1 className={`text-2xl font-bold ${textColor}`}>🏠 HomeLog</h1>
          {splitModeEnabled && (
            <span className="px-3 py-1 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 text-sm rounded-full font-medium">
              Split Mode ON
            </span>
          )}
        </div>
        
        <div className="flex items-center gap-4">
          {balance !== 0 && splitModeEnabled && (
            <div className={`px-3 py-1 rounded-lg ${balance > 0 ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400' : 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400'} text-sm font-medium`}>
              {balance > 0 ? `+${balance.toFixed(2)}` : balance.toFixed(2)} €
            </div>
          )}
          <button 
            onClick={() => setDarkMode(!darkMode)}
            className={`p-2 rounded-full hover:bg-gray-100 ${darkMode ? 'hover:bg-gray-700' : ''}`}
          >
            {darkMode ? '☀️' : '🌙'}
          </button>
        </div>
      </div>
    </nav>
  );

  const Sidebar = () => {
    const menuItems = [
      { id: 'dashboard', icon: Home, label: 'Dashboard' },
      { id: 'settlements', icon: ArrowLeftRight, label: 'Pagamenti', badge: balance !== 0 ? '!' : null },
      { id: 'settings', icon: Settings, label: 'Impostazioni' },
    ];

    return (
      <aside className={`${cardBg} border-r ${borderColor} w-64 p-6 hidden lg:block sticky top-16 h-[calc(100vh-4rem)]`}>
        <nav className="space-y-2">
          {menuItems.map(item => (
            <button
              key={item.id}
              onClick={() => setCurrentView(item.id)}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-colors relative ${
                currentView === item.id 
                  ? 'bg-blue-500 text-white' 
                  : `${textColor} hover:bg-gray-100 ${darkMode ? 'hover:bg-gray-700' : ''}`
              }`}
            >
              <item.icon size={20} />
              <span>{item.label}</span>
              {item.badge && (
                <span className="absolute right-3 top-1/2 -translate-y-1/2 w-2 h-2 bg-red-500 rounded-full animate-pulse" />
              )}
            </button>
          ))}
        </nav>
      </aside>
    );
  };

  const MobileNav = () => {
    const menuItems = [
      { id: 'dashboard', icon: Home, label: 'Home' },
      { id: 'settlements', icon: ArrowLeftRight, label: 'Pagamenti' },
      { id: 'settings', icon: Settings, label: 'Altro' },
    ];

    return (
      <nav className={`lg:hidden fixed bottom-0 left-0 right-0 ${cardBg} border-t ${borderColor} px-4 py-2 z-50`}>
        <div className="flex items-center justify-around">
          {menuItems.map(item => (
            <button
              key={item.id}
              onClick={() => setCurrentView(item.id)}
              className={`flex flex-col items-center gap-1 py-2 px-4 rounded-lg relative ${
                currentView === item.id ? 'text-blue-500' : textSecondary
              }`}
            >
              <item.icon size={24} />
              <span className="text-xs">{item.label}</span>
              {item.id === 'settlements' && balance !== 0 && (
                <span className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full" />
              )}
            </button>
          ))}
        </div>
      </nav>
    );
  };

  const FAB = () => (
    <button 
      onClick={() => setShowAddExpense(true)}
      className="fixed bottom-24 lg:bottom-8 right-8 w-14 h-14 bg-blue-500 text-white rounded-full shadow-lg hover:bg-blue-600 flex items-center justify-center z-40"
    >
      <Plus size={24} />
    </button>
  );

  return (
    <div className={`min-h-screen ${bgColor} ${textColor}`}>
      <Navbar />
      
      <div className="flex max-w-[1920px] mx-auto">
        <Sidebar />
        
        <main className="flex-1 p-6 pb-24 lg:pb-6">
          {currentView === 'dashboard' && <DashboardView />}
          {currentView === 'settlements' && <SettlementsView />}
          {currentView === 'settings' && <SettingsView />}
        </main>
      </div>

      <MobileNav />
      <FAB />
      {showAddExpense && <AddExpenseModal />}
      {showSettlement && <SettlementModal />}
    </div>
  );
};

export default HomeLogSplitPrototype;
