import React, { useState } from 'react';
import { Home, Receipt, TrendingUp, Droplet, Settings, Plus, Menu, X, Sun, Moon, Zap, Flame, Trash2, FileText, Calendar, Euro, ChevronRight, Filter, Bell, User, DollarSign, ArrowLeftRight, Check, Users } from 'lucide-react';
import { LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

// Mock Data
const currentUser = { id: 1, name: 'Simone', email: 's.girardi92@gmail.com' };
const familyMembers = [
  { id: 1, name: 'Simone', email: 's.girardi92@gmail.com', role: 'admin' },
  { id: 2, name: 'Marina', email: 'marina@example.com', role: 'user' }
];

// Strategia famiglia: 'split' o 'shared'
const familyStrategy = 'split'; // o 'shared'

const mockExpenses = [
  { id: 1, date: '2025-01-27', category: '🍕 Alimentari', amount: 45.00, description: 'Pizzeria Da Mario', paidBy: 1, splitWith: [2], settled: false },
  { id: 2, date: '2025-01-26', category: '🏠 Casa', amount: 58.00, description: 'Bolletta Luce E.ON', paidBy: 1, splitWith: [2], settled: false },
  { id: 3, date: '2025-01-25', category: '🚗 Trasporti', amount: 60.00, description: 'Rifornimento Esso', paidBy: 2, splitWith: [1], settled: false },
  { id: 4, date: '2025-01-20', category: '🏠 Casa', amount: 160.00, description: 'Bolletta Gas E.ON', paidBy: 1, splitWith: [2], settled: false },
  { id: 5, date: '2025-01-15', category: '🎬 Intrattenimento', amount: 24.00, description: 'Cinema Multisala', paidBy: 2, splitWith: [1], settled: true },
];

// Transazioni di saldo
const mockSettlements = [
  { id: 1, date: '2025-01-22', from: 2, to: 1, amount: 50.00, note: 'Bonifico saldo parziale' },
  { id: 2, date: '2025-01-10', from: 1, to: 2, amount: 30.00, note: 'Saldo spese dicembre' },
];

const mockUtilities = [
  { id: 1, name: 'Luce', provider: 'E.ON Energia', icon: Zap, color: 'yellow', lastBill: { amount: 58.00, date: '07/04/2025' }, consumption: '143 kWh' },
  { id: 2, name: 'Gas', provider: 'E.ON Energia', icon: Flame, color: 'orange', lastBill: { amount: 160.00, date: '13/04/2025' }, consumption: '121 Smc' },
  { id: 3, name: 'Acqua', provider: 'ETRA', icon: Droplet, color: 'cyan', lastBill: { amount: 81.58, date: '07/04/2025' }, consumption: '35 mc' },
  { id: 4, name: 'Rifiuti', provider: 'ETRA', icon: Trash2, color: 'green', lastBill: { amount: 351.32, date: '16/05/2025' }, consumption: '130 mq' },
];

const chartData = [
  { month: 'Lug', amount: 823 },
  { month: 'Ago', amount: 856 },
  { month: 'Set', amount: 902 },
  { month: 'Ott', amount: 980 },
  { month: 'Nov', amount: 1050 },
  { month: 'Dic', amount: 1245 },
];

const pieData = [
  { name: 'Casa', value: 450, color: '#3B82F6' },
  { name: 'Alimentari', value: 320, color: '#10B981' },
  { name: 'Trasporti', value: 240, color: '#F59E0B' },
  { name: 'Intrattenimento', value: 150, color: '#8B5CF6' },
  { name: 'Altro', value: 85, color: '#6B7280' },
];

// Calcola bilancio
const calculateBalance = () => {
  let balance = 0;
  
  mockExpenses.forEach(expense => {
    if (!expense.settled && expense.splitWith.length > 0) {
      const splitAmount = expense.amount / (expense.splitWith.length + 1);
      
      if (expense.paidBy === currentUser.id) {
        // Simone ha pagato, Marina deve
        balance += splitAmount * expense.splitWith.length;
      } else if (expense.splitWith.includes(currentUser.id)) {
        // Marina ha pagato, Simone deve
        balance -= expense.amount / (expense.splitWith.length + 1);
      }
    }
  });
  
  // Sottrai bonifici
  mockSettlements.forEach(settlement => {
    if (settlement.from === currentUser.id) {
      balance += settlement.amount;
    } else if (settlement.to === currentUser.id) {
      balance -= settlement.amount;
    }
  });
  
  return balance;
};

const HomeLogPrototype = () => {
  const [currentView, setCurrentView] = useState('dashboard');
  const [darkMode, setDarkMode] = useState(false);
  const [selectedProperty, setSelectedProperty] = useState('Padova - Via Roma 71/B');
  const [showAddExpense, setShowAddExpense] = useState(false);
  const [showSettleModal, setShowSettleModal] = useState(false);

  const bgColor = darkMode ? 'bg-gray-900' : 'bg-gray-50';
  const cardBg = darkMode ? 'bg-gray-800' : 'bg-white';
  const textColor = darkMode ? 'text-gray-100' : 'text-gray-900';
  const textSecondary = darkMode ? 'text-gray-400' : 'text-gray-600';
  const borderColor = darkMode ? 'border-gray-700' : 'border-gray-200';

  const balance = calculateBalance();
  const otherUser = familyMembers.find(m => m.id !== currentUser.id);

  const getUserName = (userId) => {
    const user = familyMembers.find(u => u.id === userId);
    return user ? user.name : 'Unknown';
  };

  // Components
  const Navbar = () => (
    <nav className={`${darkMode ? 'bg-gray-800 border-gray-700' : 'bg-white border-gray-200'} border-b px-6 py-4 sticky top-0 z-50`}>
      <div className="flex items-center justify-between max-w-7xl mx-auto">
        <div className="flex items-center gap-4">
          <h1 className={`text-2xl font-bold ${textColor}`}>🏠 HomeLog</h1>
        </div>
        
        <div className="flex items-center gap-4">
          <select 
            className={`${cardBg} ${textColor} border ${borderColor} rounded-lg px-3 py-2 text-sm hidden md:block`}
            value={selectedProperty}
            onChange={(e) => setSelectedProperty(e.target.value)}
          >
            <option>Padova - Via Roma 71/B</option>
            <option>Genova Pontedecimo</option>
          </select>
          
          <button className={`p-2 rounded-full hover:bg-gray-100 ${darkMode ? 'hover:bg-gray-700' : ''}`}>
            <Bell className={textColor} size={20} />
          </button>
          
          <button 
            onClick={() => setDarkMode(!darkMode)}
            className={`p-2 rounded-full hover:bg-gray-100 ${darkMode ? 'hover:bg-gray-700' : ''}`}
          >
            {darkMode ? <Sun className={textColor} size={20} /> : <Moon className={textColor} size={20} />}
          </button>
          
          <button className={`p-2 rounded-full hover:bg-gray-100 ${darkMode ? 'hover:bg-gray-700' : ''}`}>
            <User className={textColor} size={20} />
          </button>
        </div>
      </div>
    </nav>
  );

  const Sidebar = () => {
    const menuItems = [
      { id: 'dashboard', icon: Home, label: 'Dashboard' },
      { id: 'expenses', icon: Receipt, label: 'Spese' },
      { id: 'balance', icon: ArrowLeftRight, label: 'Bilancio', badge: balance !== 0 },
      { id: 'utilities', icon: Droplet, label: 'Utilities' },
      { id: 'settings', icon: Settings, label: 'Impostazioni' },
    ];

    return (
      <aside className={`${cardBg} border-r ${borderColor} w-64 p-6 hidden lg:block sticky top-16 h-[calc(100vh-4rem)]`}>
        <nav className="space-y-2">
          {menuItems.map(item => (
            <button
              key={item.id}
              onClick={() => setCurrentView(item.id)}
              className={`w-full flex items-center justify-between gap-3 px-4 py-3 rounded-lg transition-colors ${
                currentView === item.id 
                  ? 'bg-blue-500 text-white' 
                  : `${textColor} hover:bg-gray-100 ${darkMode ? 'hover:bg-gray-700' : ''}`
              }`}
            >
              <div className="flex items-center gap-3">
                <item.icon size={20} />
                <span>{item.label}</span>
              </div>
              {item.badge && Math.abs(balance) > 0 && (
                <span className="bg-red-500 text-white text-xs rounded-full px-2 py-0.5">!</span>
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
      { id: 'expenses', icon: Receipt, label: 'Spese' },
      { id: 'balance', icon: ArrowLeftRight, label: 'Bilancio' },
      { id: 'utilities', icon: Droplet, label: 'Utilities' },
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
              {item.id === 'balance' && Math.abs(balance) > 0 && (
                <span className="absolute top-1 right-2 bg-red-500 w-2 h-2 rounded-full"></span>
              )}
            </button>
          ))}
        </div>
      </nav>
    );
  };

  const DashboardView = () => (
    <div className="space-y-6">
      {/* KPI Cards - con Bilancio */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Bilancio Card - NUOVA */}
        {familyStrategy === 'split' && (
          <div className={`${cardBg} rounded-xl p-6 border ${borderColor} backdrop-blur-xl bg-opacity-70 ${Math.abs(balance) > 50 ? 'ring-2 ring-orange-500' : ''}`}>
            <div className="flex items-center justify-between mb-2">
              <span className={`${textSecondary} text-sm font-medium`}>Bilancio Famiglia</span>
              <ArrowLeftRight className={balance > 0 ? 'text-green-500' : balance < 0 ? 'text-red-500' : 'text-gray-500'} size={20} />
            </div>
            <div className={`text-3xl font-bold mb-1 ${balance > 0 ? 'text-green-600' : balance < 0 ? 'text-red-600' : textColor}`}>
              {balance > 0 ? '+' : ''}{balance.toFixed(2)}€
            </div>
            <div className={`text-sm mb-3 ${balance > 0 ? 'text-green-600' : balance < 0 ? 'text-red-600' : textSecondary}`}>
              {balance > 0 
                ? `${otherUser.name} ti deve` 
                : balance < 0 
                ? `Devi a ${otherUser.name}` 
                : 'Tutto saldato'}
            </div>
            {Math.abs(balance) > 0 && (
              <button 
                onClick={() => setShowSettleModal(true)}
                className="w-full mt-2 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors text-sm font-medium"
              >
                Salda
              </button>
            )}
          </div>
        )}

        {/* Spese Mese */}
        <div className={`${cardBg} rounded-xl p-6 border ${borderColor} backdrop-blur-xl bg-opacity-70`}>
          <div className="flex items-center justify-between mb-2">
            <span className={`${textSecondary} text-sm font-medium`}>Spese Mese</span>
            <Euro className="text-blue-500" size={20} />
          </div>
          <div className={`text-3xl font-bold ${textColor} mb-1`}>€1.245,50</div>
          <div className="text-sm text-green-500 mb-3">+12% vs mese scorso</div>
          
          <div className={`pt-3 border-t ${borderColor}`}>
            <div className={`text-xs ${textSecondary} mb-2`}>Ultima spesa:</div>
            <div className="flex items-center justify-between">
              <div>
                <div className={`text-sm font-medium ${textColor}`}>Pizzeria Da Mario</div>
                <div className={`text-xs ${textSecondary}`}>27/01 • Pagato da {getUserName(1)}</div>
              </div>
              <div className="text-sm font-bold text-blue-500">€45,00</div>
            </div>
          </div>
        </div>

        {/* Budget */}
        <div className={`${cardBg} rounded-xl p-6 border ${borderColor} backdrop-blur-xl bg-opacity-70`}>
          <div className="flex items-center justify-between mb-2">
            <span className={`${textSecondary} text-sm font-medium`}>Budget</span>
            <TrendingUp className="text-green-500" size={20} />
          </div>
          <div className={`text-3xl font-bold ${textColor} mb-1`}>75%</div>
          <div className="text-sm text-orange-500 mb-3">€250 rimanenti</div>
          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
            <div className="bg-gradient-to-r from-green-500 to-orange-500 h-2 rounded-full" style={{ width: '75%' }} />
          </div>
        </div>
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className={`${cardBg} rounded-xl p-6 border ${borderColor} backdrop-blur-xl bg-opacity-70`}>
          <h3 className={`text-lg font-semibold ${textColor} mb-4`}>📈 Spese Mensili</h3>
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke={darkMode ? '#374151' : '#E5E7EB'} />
              <XAxis dataKey="month" stroke={darkMode ? '#9CA3AF' : '#6B7280'} />
              <YAxis stroke={darkMode ? '#9CA3AF' : '#6B7280'} />
              <Tooltip contentStyle={{ backgroundColor: darkMode ? '#1F2937' : '#FFFFFF', border: `1px solid ${darkMode ? '#374151' : '#E5E7EB'}`, borderRadius: '8px' }} />
              <Bar dataKey="amount" fill="#3B82F6" radius={[8, 8, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className={`${cardBg} rounded-xl p-6 border ${borderColor} backdrop-blur-xl bg-opacity-70`}>
          <h3 className={`text-lg font-semibold ${textColor} mb-4`}>🏠 Spese per Categoria</h3>
          <ResponsiveContainer width="100%" height={250}>
            <PieChart>
              <Pie data={pieData} cx="50%" cy="50%" outerRadius={80} fill="#8884d8" dataKey="value" label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}>
                {pieData.map((entry, index) => (<Cell key={`cell-${index}`} fill={entry.color} />))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );

  const BalanceView = () => (
    <div className="space-y-6">
      {/* Header con strategia */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className={`text-2xl font-bold ${textColor}`}>Bilancio Famiglia</h2>
          <p className={textSecondary}>Strategia: {familyStrategy === 'split' ? '✂️ Divisione equa 50/50' : '🏦 Conto comune'}</p>
        </div>
        <button 
          onClick={() => setCurrentView('settings')}
          className="px-4 py-2 bg-gray-200 dark:bg-gray-700 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
        >
          Modifica strategia
        </button>
      </div>

      {familyStrategy === 'split' && (
        <>
          {/* Bilancio Totale */}
          <div className={`${cardBg} rounded-xl p-8 border ${borderColor} backdrop-blur-xl bg-opacity-70 text-center`}>
            <div className={`text-sm ${textSecondary} mb-2`}>Bilancio corrente</div>
            <div className={`text-5xl font-bold mb-4 ${balance > 0 ? 'text-green-600' : balance < 0 ? 'text-red-600' : textColor}`}>
              {balance > 0 ? '+' : ''}{balance.toFixed(2)}€
            </div>
            <div className={`text-lg mb-6 ${balance > 0 ? 'text-green-600' : balance < 0 ? 'text-red-600' : textSecondary}`}>
              {balance > 0 
                ? `${otherUser.name} ti deve ${Math.abs(balance).toFixed(2)}€` 
                : balance < 0 
                ? `Devi a ${otherUser.name} ${Math.abs(balance).toFixed(2)}€` 
                : '✅ Tutto saldato!'}
            </div>
            {Math.abs(balance) > 0 && (
              <button 
                onClick={() => setShowSettleModal(true)}
                className="px-8 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors font-semibold"
              >
                Salda adesso
              </button>
            )}
          </div>

          {/* Spese non saldate */}
          <div className={`${cardBg} rounded-xl p-6 border ${borderColor} backdrop-blur-xl bg-opacity-70`}>
            <h3 className={`text-lg font-semibold ${textColor} mb-4`}>💳 Spese da saldare</h3>
            <div className="space-y-3">
              {mockExpenses.filter(e => !e.settled && e.splitWith.length > 0).map(expense => {
                const splitAmount = expense.amount / (expense.splitWith.length + 1);
                const isPayer = expense.paidBy === currentUser.id;
                
                return (
                  <div key={expense.id} className={`p-4 rounded-lg border ${borderColor} ${isPayer ? 'bg-green-50 dark:bg-green-900/20' : 'bg-red-50 dark:bg-red-900/20'}`}>
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <span className={`text-sm ${textSecondary}`}>{expense.date}</span>
                          <span className="text-sm font-medium">{expense.category}</span>
                        </div>
                        <div className={`font-medium ${textColor}`}>{expense.description}</div>
                        <div className={`text-sm ${textSecondary} mt-1`}>
                          Pagato da {getUserName(expense.paidBy)} • Split: €{splitAmount.toFixed(2)}
                        </div>
                      </div>
                      <div className="text-right">
                        <div className={`font-bold text-lg ${isPayer ? 'text-green-600' : 'text-red-600'}`}>
                          {isPayer ? '+' : '-'}€{splitAmount.toFixed(2)}
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Storia movimenti */}
          <div className={`${cardBg} rounded-xl p-6 border ${borderColor} backdrop-blur-xl bg-opacity-70`}>
            <h3 className={`text-lg font-semibold ${textColor} mb-4`}>🔄 Storia Movimenti</h3>
            <div className="space-y-3">
              {mockSettlements.map(settlement => (
                <div key={settlement.id} className={`p-4 rounded-lg border ${borderColor} bg-blue-50 dark:bg-blue-900/20`}>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="p-2 bg-blue-500 rounded-full">
                        <ArrowLeftRight className="text-white" size={20} />
                      </div>
                      <div>
                        <div className={`font-medium ${textColor}`}>
                          {getUserName(settlement.from)} → {getUserName(settlement.to)}
                        </div>
                        <div className={`text-sm ${textSecondary}`}>{settlement.date}</div>
                        {settlement.note && (
                          <div className={`text-xs ${textSecondary} italic`}>{settlement.note}</div>
                        )}
                      </div>
                    </div>
                    <div className="font-bold text-blue-600">€{settlement.amount.toFixed(2)}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </>
      )}

      {familyStrategy === 'shared' && (
        <div className={`${cardBg} rounded-xl p-8 border ${borderColor} backdrop-blur-xl bg-opacity-70 text-center`}>
          <div className="text-6xl mb-4">🏦</div>
          <div className={`text-xl font-semibold ${textColor} mb-2`}>Conto comune attivo</div>
          <div className={textSecondary}>Le spese vengono solo tracciate, non vengono calcolati debiti</div>
        </div>
      )}
    </div>
  );

  const SettleModal = () => {
    if (!showSettleModal) return null;
    
    const [amount, setAmount] = useState(Math.abs(balance).toFixed(2));
    const [note, setNote] = useState('');
    
    return (
      <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={() => setShowSettleModal(false)}>
        <div className={`${cardBg} rounded-2xl p-6 max-w-md w-full`} onClick={e => e.stopPropagation()}>
          <div className="flex items-center justify-between mb-6">
            <h3 className={`text-xl font-bold ${textColor}`}>Salda Conto</h3>
            <button onClick={() => setShowSettleModal(false)}>
              <X className={textColor} />
            </button>
          </div>

          <div className={`p-4 rounded-lg mb-6 ${balance > 0 ? 'bg-green-50 dark:bg-green-900/20' : 'bg-red-50 dark:bg-red-900/20'}`}>
            <div className={`text-sm ${textSecondary} mb-2`}>
              {balance > 0 ? `${otherUser.name} ti deve:` : `Devi a ${otherUser.name}:`}
            </div>
            <div className={`text-3xl font-bold ${balance > 0 ? 'text-green-600' : 'text-red-600'}`}>
              €{Math.abs(balance).toFixed(2)}
            </div>
          </div>

          <div className="space-y-4">
            <div>
              <label className={`block text-sm ${textSecondary} mb-1`}>Importo da saldare</label>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">€</span>
                <input
                  type="number"
                  step="0.01"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  className={`w-full pl-8 pr-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
                />
              </div>
            </div>

            <div>
              <label className={`block text-sm ${textSecondary} mb-1`}>Nota (opzionale)</label>
              <input
                type="text"
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="Es: Bonifico gennaio"
                className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
              />
            </div>

            <div className={`p-3 rounded-lg bg-blue-50 dark:bg-blue-900/20 text-sm ${textColor}`}>
              <strong>Conferma:</strong> Stai registrando un {balance > 0 ? 'bonifico ricevuto' : 'bonifico inviato'} di €{amount} {balance > 0 ? 'da' : 'a'} {otherUser.name}
            </div>
          </div>

          <div className="flex gap-3 mt-6">
            <button 
              onClick={() => setShowSettleModal(false)}
              className={`flex-1 px-4 py-3 border ${borderColor} rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 ${textColor}`}
            >
              Annulla
            </button>
            <button 
              onClick={() => {
                alert('Bonifico registrato! (In produzione: salva nel database)');
                setShowSettleModal(false);
              }}
              className="flex-1 px-4 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600 flex items-center justify-center gap-2"
            >
              <Check size={20} />
              Conferma
            </button>
          </div>
        </div>
      </div>
    );
  };

  const AddExpenseModal = () => {
    if (!showAddExpense) return null;
    
    const [paidBy, setPaidBy] = useState(currentUser.id);
    const [splitMode, setSplitMode] = useState(familyStrategy === 'split' ? 'equal' : 'none');
    
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
            {/* Chi ha pagato */}
            <div className={`p-3 rounded-lg border ${borderColor} bg-blue-50 dark:bg-blue-900/20`}>
              <label className={`block text-sm font-medium ${textColor} mb-2`}>Chi ha pagato? *</label>
              <div className="flex gap-2">
                {familyMembers.map(member => (
                  <button
                    key={member.id}
                    onClick={() => setPaidBy(member.id)}
                    className={`flex-1 px-4 py-2 rounded-lg font-medium transition-colors ${
                      paidBy === member.id
                        ? 'bg-blue-500 text-white'
                        : `${cardBg} border ${borderColor} ${textColor} hover:bg-gray-50 dark:hover:bg-gray-700`
                    }`}
                  >
                    {member.name}
                  </button>
                ))}
              </div>
            </div>

            {/* Divisione */}
            {familyStrategy === 'split' && (
              <div className={`p-3 rounded-lg border ${borderColor}`}>
                <label className={`block text-sm font-medium ${textColor} mb-2`}>Divisione</label>
                <select
                  value={splitMode}
                  onChange={(e) => setSplitMode(e.target.value)}
                  className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
                >
                  <option value="equal">✂️ Dividi equamente</option>
                  <option value="none">👤 Solo chi ha pagato</option>
                  <option value="custom">⚙️ Personalizzato</option>
                </select>
                {splitMode === 'equal' && (
                  <div className={`mt-2 text-xs ${textSecondary}`}>
                    La spesa verrà divisa automaticamente tra tutti i membri della famiglia
                  </div>
                )}
              </div>
            )}

            <div>
              <label className={`block text-sm ${textSecondary} mb-1`}>Importo *</label>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">€</span>
                <input 
                  type="number" 
                  step="0.01"
                  className={`w-full pl-8 pr-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
                  placeholder="0,00"
                />
              </div>
            </div>

            <div>
              <label className={`block text-sm ${textSecondary} mb-1`}>Categoria *</label>
              <select className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
                <option>🍕 Alimentari e Ristorazione</option>
                <option>🏠 Casa</option>
                <option>🚗 Trasporti</option>
                <option>🎬 Intrattenimento</option>
              </select>
            </div>

            <div>
              <label className={`block text-sm ${textSecondary} mb-1`}>Data *</label>
              <input 
                type="date" 
                className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
                defaultValue="2025-01-27"
              />
            </div>

            <div>
              <label className={`block text-sm ${textSecondary} mb-1`}>Descrizione</label>
              <textarea 
                className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}
                rows="3"
                placeholder="Es: Cena al ristorante..."
              />
            </div>
          </div>

          <div className="flex gap-3 mt-6">
            <button 
              onClick={() => setShowAddExpense(false)}
              className={`flex-1 px-4 py-3 border ${borderColor} rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 ${textColor}`}
            >
              Annulla
            </button>
            <button 
              onClick={() => {
                alert('Spesa aggiunta! (In produzione: salva nel database)');
                setShowAddExpense(false);
              }}
              className="flex-1 px-4 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600"
            >
              Salva
            </button>
          </div>
        </div>
      </div>
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
          {currentView === 'balance' && <BalanceView />}
          {currentView === 'utilities' && <div className="text-center py-20"><div className="text-4xl mb-4">💡</div><div className={textColor}>Vista Utilities (già implementata)</div></div>}
          {currentView === 'settings' && <div className="text-center py-20"><div className="text-4xl mb-4">⚙️</div><div className={textColor}>Vista Impostazioni (già implementata)</div></div>}
        </main>
      </div>

      <MobileNav />
      <FAB />
      <AddExpenseModal />
      <SettleModal />
    </div>
  );
};

export default HomeLogPrototype;
