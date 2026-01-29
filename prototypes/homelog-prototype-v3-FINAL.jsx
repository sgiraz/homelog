import React, { useState } from 'react';
import { Home, Receipt, TrendingUp, Droplet, Settings, Plus, Menu, X, Sun, Moon, Zap, Flame, Trash2, FileText, Calendar, Euro, ChevronRight, Filter, Bell, User } from 'lucide-react';
import { LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

// Mock Data
const mockExpenses = [
  { id: 1, date: '2025-01-27', category: '🍕 Alimentari', subcategory: 'Ristoranti', amount: 45.00, description: 'Pizzeria Da Mario', property: 'Padova' },
  { id: 2, date: '2025-01-26', category: '🏠 Casa', subcategory: 'Utenze', amount: 58.00, description: 'Bolletta Luce E.ON', property: 'Padova' },
  { id: 3, date: '2025-01-25', category: '🚗 Trasporti', subcategory: 'Carburante', amount: 60.00, description: 'Rifornimento Esso', property: 'Padova' },
  { id: 4, date: '2025-01-20', category: '🏠 Casa', subcategory: 'Utenze', amount: 160.00, description: 'Bolletta Gas E.ON', property: 'Padova' },
  { id: 5, date: '2025-01-15', category: '🎬 Intrattenimento', subcategory: 'Cinema', amount: 24.00, description: 'Cinema Multisala', property: 'Padova' },
];

const mockUtilities = [
  { id: 1, name: 'Luce', provider: 'E.ON Energia', icon: Zap, color: 'yellow', lastReading: '27/01/2025', lastBill: { amount: 58.00, date: '07/04/2025' }, consumption: '143 kWh', alert: null },
  { id: 2, name: 'Gas', provider: 'E.ON Energia', icon: Flame, color: 'orange', lastReading: '22/12/2024', lastBill: { amount: 160.00, date: '13/04/2025' }, consumption: '121 Smc', alert: 'Autolettura consigliata tra 5 giorni' },
  { id: 3, name: 'Acqua', provider: 'ETRA', icon: Droplet, color: 'cyan', lastReading: '22/12/2024', lastBill: { amount: 81.58, date: '07/04/2025' }, consumption: '35 mc', alert: null },
  { id: 4, name: 'Rifiuti', provider: 'ETRA', icon: Trash2, color: 'green', lastReading: null, lastBill: { amount: 351.32, date: '16/05/2025' }, consumption: '130 mq', alert: null },
];

const mockProjects = [
  { id: 1, name: '🏠 Ristrutturazione Bagno 2025', budget: 5000, spent: 2500, startDate: '15/01/2025', endDate: '28/02/2025', property: 'Padova', expenseCount: 8 },
  { id: 2, name: '✈️ Viaggio Giappone 2025', budget: 8000, spent: 1200, startDate: '01/09/2025', endDate: '20/09/2025', property: null, expenseCount: 3 },
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

const HomeLogPrototype = () => {
  const [currentView, setCurrentView] = useState('dashboard');
  const [darkMode, setDarkMode] = useState(false);
  const [selectedProperty, setSelectedProperty] = useState('Padova - Via Roma 71/B');
  const [showMobileMenu, setShowMobileMenu] = useState(false);
  const [showAddExpense, setShowAddExpense] = useState(false);

  const bgColor = darkMode ? 'bg-gray-900' : 'bg-gray-50';
  const cardBg = darkMode ? 'bg-gray-800' : 'bg-white';
  const textColor = darkMode ? 'text-gray-100' : 'text-gray-900';
  const textSecondary = darkMode ? 'text-gray-400' : 'text-gray-600';
  const borderColor = darkMode ? 'border-gray-700' : 'border-gray-200';

  // Components
  const Navbar = () => (
    <nav className={`${darkMode ? 'bg-gray-800 border-gray-700' : 'bg-white border-gray-200'} border-b px-6 py-4 sticky top-0 z-50`}>
      <div className="flex items-center justify-between max-w-7xl mx-auto">
        <div className="flex items-center gap-4">
          <button 
            className="lg:hidden"
            onClick={() => setShowMobileMenu(!showMobileMenu)}
          >
            {showMobileMenu ? <X className={textColor} /> : <Menu className={textColor} />}
          </button>
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
      { id: 'projects', icon: TrendingUp, label: 'Progetti' },
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
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                currentView === item.id 
                  ? 'bg-blue-500 text-white' 
                  : `${textColor} hover:bg-gray-100 ${darkMode ? 'hover:bg-gray-700' : ''}`
              }`}
            >
              <item.icon size={20} />
              <span>{item.label}</span>
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
      { id: 'projects', icon: TrendingUp, label: 'Progetti' },
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
              className={`flex flex-col items-center gap-1 py-2 px-4 rounded-lg ${
                currentView === item.id ? 'text-blue-500' : textSecondary
              }`}
            >
              <item.icon size={24} />
              <span className="text-xs">{item.label}</span>
            </button>
          ))}
        </div>
      </nav>
    );
  };

  const DashboardView = () => {
    // Filtri unificati
    const [timePreset, setTimePreset] = useState('6m');
    const [dateFrom, setDateFrom] = useState('2024-07-01');
    const [dateTo, setDateTo] = useState('2024-12-31');
    const [selectedCategory, setSelectedCategory] = useState('all');
    const [useCustomRange, setUseCustomRange] = useState(false);

    // Calcola periodo label
    const getPeriodLabel = () => {
      if (useCustomRange) {
        return `${dateFrom} - ${dateTo}`;
      }
      const labels = {
        '1d': 'Oggi',
        '1m': 'Ultimo mese',
        '3m': 'Ultimi 3 mesi',
        '6m': 'Ultimi 6 mesi',
        '1y': 'Ultimo anno'
      };
      return labels[timePreset] || 'Periodo personalizzato';
    };

    // Filtra dati grafici (simulazione - in produzione sarà una chiamata API)
    const getFilteredChartData = () => {
      // Simula filtro categoria
      if (selectedCategory !== 'all') {
        // In produzione: filtrare i dati reali
        return chartData.map(d => ({ ...d, amount: d.amount * 0.6 })); // Mock: riduco del 40%
      }
      return chartData;
    };

    const getFilteredPieData = () => {
      if (selectedCategory !== 'all') {
        return pieData.filter(d => d.name.toLowerCase() === selectedCategory.toLowerCase());
      }
      return pieData;
    };

    const filteredBarData = getFilteredChartData();
    const filteredPieData = getFilteredPieData();
    
    return (
      <div className="space-y-6">
        {/* KPI Cards - Desktop: 3 colonne, Mobile: 1 colonna */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Spese Mese - con Ultima Spesa */}
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
                  <div className={`text-xs ${textSecondary}`}>27/01 • Alimentari</div>
                </div>
                <div className="text-sm font-bold text-blue-500">€45,00</div>
              </div>
              <button 
                onClick={() => setCurrentView('expenses')}
                className={`mt-2 text-xs text-blue-500 hover:text-blue-600 font-medium flex items-center gap-1`}
              >
                Vedi tutte <ChevronRight size={14} />
              </button>
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

          {/* Alert */}
          <div className={`${cardBg} rounded-xl p-6 border ${borderColor} backdrop-blur-xl bg-opacity-70`}>
            <div className="flex items-center justify-between mb-2">
              <span className={`${textSecondary} text-sm font-medium`}>Alert</span>
              <Bell className="text-orange-500" size={20} />
            </div>
            <div className={`text-3xl font-bold ${textColor} mb-1`}>3</div>
            <div className="text-sm text-orange-500">Richiede attenzione</div>
          </div>
        </div>

        {/* FILTRI UNIFICATI */}
        <div className={`${cardBg} rounded-xl p-4 border ${borderColor} backdrop-blur-xl bg-opacity-70`}>
          <div className="flex flex-col lg:flex-row gap-4 items-start lg:items-center">
            <div className="flex items-center gap-2">
              <Filter className={textSecondary} size={20} />
              <span className={`font-medium ${textColor}`}>Filtri Grafici</span>
            </div>

            <div className="flex-1 flex flex-wrap gap-3">
              {/* Preset Temporali */}
              <div className="flex gap-2">
                {[
                  { value: '1d', label: 'Oggi' },
                  { value: '1m', label: '1 Mese' },
                  { value: '3m', label: '3 Mesi' },
                  { value: '6m', label: '6 Mesi' },
                  { value: '1y', label: '1 Anno' }
                ].map(preset => (
                  <button
                    key={preset.value}
                    onClick={() => {
                      setTimePreset(preset.value);
                      setUseCustomRange(false);
                    }}
                    className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                      !useCustomRange && timePreset === preset.value
                        ? 'bg-blue-500 text-white'
                        : `${cardBg} border ${borderColor} ${textSecondary} hover:bg-gray-50 dark:hover:bg-gray-700`
                    }`}
                  >
                    {preset.label}
                  </button>
                ))}
              </div>

              {/* Separatore */}
              <div className={`h-8 w-px bg-gray-300 dark:bg-gray-600`} />

              {/* Range Personalizzato */}
              <div className="flex items-center gap-2">
                <input
                  type="date"
                  value={dateFrom}
                  onChange={(e) => {
                    setDateFrom(e.target.value);
                    setUseCustomRange(true);
                  }}
                  className={`px-3 py-1.5 border ${borderColor} rounded-lg text-sm ${cardBg} ${textColor}`}
                />
                <span className={textSecondary}>→</span>
                <input
                  type="date"
                  value={dateTo}
                  onChange={(e) => {
                    setDateTo(e.target.value);
                    setUseCustomRange(true);
                  }}
                  className={`px-3 py-1.5 border ${borderColor} rounded-lg text-sm ${cardBg} ${textColor}`}
                />
              </div>

              {/* Separatore */}
              <div className={`h-8 w-px bg-gray-300 dark:bg-gray-600`} />

              {/* Filtro Categoria */}
              <select
                value={selectedCategory}
                onChange={(e) => setSelectedCategory(e.target.value)}
                className={`px-3 py-1.5 border ${borderColor} rounded-lg text-sm ${cardBg} ${textColor}`}
              >
                <option value="all">📁 Tutte le categorie</option>
                <option value="casa">🏠 Casa</option>
                <option value="alimentari">🍕 Alimentari</option>
                <option value="trasporti">🚗 Trasporti</option>
                <option value="intrattenimento">🎬 Intrattenimento</option>
                <option value="altro">📦 Altro</option>
              </select>
            </div>

            {/* Periodo Attivo */}
            <div className={`text-sm ${textSecondary} whitespace-nowrap`}>
              {getPeriodLabel()}
            </div>
          </div>
        </div>

        {/* Charts - Desktop: 2 colonne grandi + 1 sidebar, Mobile: stack */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Grafico Spese Mensili - 2 colonne su desktop */}
          <div className={`lg:col-span-2 ${cardBg} rounded-xl p-6 border ${borderColor} backdrop-blur-xl bg-opacity-70`}>
            <div className="flex items-center justify-between mb-4">
              <h3 className={`text-lg font-semibold ${textColor}`}>📈 Spese Mensili</h3>
            </div>
            
            {filteredBarData.length === 0 ? (
              <div className="h-[300px] flex flex-col items-center justify-center">
                <div className={`text-6xl mb-4`}>📊</div>
                <div className={`text-lg font-medium ${textColor} mb-2`}>Nessun dato disponibile</div>
                <div className={`text-sm ${textSecondary}`}>Prova a modificare i filtri</div>
              </div>
            ) : (
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={filteredBarData}>
                  <CartesianGrid strokeDasharray="3 3" stroke={darkMode ? '#374151' : '#E5E7EB'} />
                  <XAxis dataKey="month" stroke={darkMode ? '#9CA3AF' : '#6B7280'} />
                  <YAxis stroke={darkMode ? '#9CA3AF' : '#6B7280'} />
                  <Tooltip 
                    contentStyle={{ 
                      backgroundColor: darkMode ? '#1F2937' : '#FFFFFF',
                      border: `1px solid ${darkMode ? '#374151' : '#E5E7EB'}`,
                      borderRadius: '8px'
                    }}
                  />
                  <Bar dataKey="amount" fill="#3B82F6" radius={[8, 8, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </div>

          {/* Grafico Categorie - 1 colonna su desktop */}
          <div className={`${cardBg} rounded-xl p-6 border ${borderColor} backdrop-blur-xl bg-opacity-70`}>
            <div className="flex items-center justify-between mb-4">
              <h3 className={`text-lg font-semibold ${textColor}`}>🏠 Categorie</h3>
            </div>
            
            {filteredPieData.length === 0 ? (
              <div className="h-[240px] flex flex-col items-center justify-center">
                <div className={`text-5xl mb-3`}>🥧</div>
                <div className={`text-base font-medium ${textColor} mb-1`}>Nessun dato</div>
                <div className={`text-xs ${textSecondary}`}>Modifica filtri</div>
              </div>
            ) : (
              <ResponsiveContainer width="100%" height={240}>
                <PieChart>
                  <Pie
                    data={filteredPieData}
                    cx="50%"
                    cy="50%"
                    outerRadius={70}
                    fill="#8884d8"
                    dataKey="value"
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                  >
                    {filteredPieData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            )}
          </div>
        </div>

        {/* Prossime Scadenze - Full width */}
        <div className={`${cardBg} rounded-xl p-6 border ${borderColor} backdrop-blur-xl bg-opacity-70`}>
          <h3 className={`text-lg font-semibold ${textColor} mb-4`}>💡 Prossime Scadenze</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <div className="flex items-center justify-between p-4 bg-yellow-50 dark:bg-yellow-900/20 rounded-xl border border-yellow-200 dark:border-yellow-800">
              <div className="flex items-center gap-3">
                <Zap className="text-yellow-500" size={24} />
                <div>
                  <div className={`font-medium ${textColor}`}>Bolletta Luce</div>
                  <div className={`text-sm ${textSecondary}`}>07/04/2025</div>
                </div>
              </div>
              <div className="font-bold text-yellow-600">€58,00</div>
            </div>

            <div className="flex items-center justify-between p-4 bg-orange-50 dark:bg-orange-900/20 rounded-xl border border-orange-200 dark:border-orange-800">
              <div className="flex items-center gap-3">
                <Flame className="text-orange-500" size={24} />
                <div>
                  <div className={`font-medium ${textColor}`}>Bolletta Gas</div>
                  <div className={`text-sm ${textSecondary}`}>13/04/2025</div>
                </div>
              </div>
              <div className="font-bold text-orange-600">€160,00</div>
            </div>

            <div className="flex items-center justify-between p-4 bg-green-50 dark:bg-green-900/20 rounded-xl border border-green-200 dark:border-green-800">
              <div className="flex items-center gap-3">
                <Trash2 className="text-green-500" size={24} />
                <div>
                  <div className={`font-medium ${textColor}`}>Rifiuti ETRA</div>
                  <div className={`text-sm ${textSecondary}`}>16/05/2025</div>
                </div>
              </div>
              <div className="font-bold text-green-600">€351,32</div>
            </div>
          </div>
        </div>
      </div>
    );
  };

  const ExpensesView = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className={`text-2xl font-bold ${textColor}`}>Spese</h2>
        <button className="flex items-center gap-2 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600">
          <Filter size={20} />
          Filtra
        </button>
      </div>

      <div className={`${cardBg} rounded-xl border ${borderColor} overflow-hidden`}>
        <div className={`p-4 border-b ${borderColor}`}>
          <div className="flex gap-3 flex-wrap">
            <select className={`px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
              <option>🏠 Tutte le case</option>
              <option>Padova</option>
              <option>Genova</option>
            </select>
            <select className={`px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
              <option>📁 Tutte le categorie</option>
              <option>Casa</option>
              <option>Alimentari</option>
            </select>
            <select className={`px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
              <option>📅 Gen 2025</option>
              <option>Dic 2024</option>
            </select>
          </div>
        </div>

        <div className="divide-y divide-gray-200 dark:divide-gray-700">
          {mockExpenses.map(expense => (
            <div key={expense.id} className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <span className={`text-sm ${textSecondary}`}>{expense.date}</span>
                    <span className="text-sm font-medium">{expense.category}</span>
                    {expense.subcategory && (
                      <span className={`text-sm ${textSecondary}`}>› {expense.subcategory}</span>
                    )}
                  </div>
                  <div className={`font-medium ${textColor}`}>{expense.description}</div>
                </div>
                <div className="flex items-center gap-3">
                  <div className="text-right">
                    <div className={`font-bold ${textColor}`}>€{expense.amount.toFixed(2)}</div>
                  </div>
                  <ChevronRight className={textSecondary} size={20} />
                </div>
              </div>
            </div>
          ))}
        </div>

        <div className={`p-4 border-t ${borderColor} bg-gray-50 dark:bg-gray-800/50`}>
          <div className="flex items-center justify-between font-bold">
            <span className={textColor}>Totale periodo</span>
            <span className={textColor}>€{mockExpenses.reduce((sum, e) => sum + e.amount, 0).toFixed(2)}</span>
          </div>
        </div>
      </div>
    </div>
  );

  const UtilitiesView = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className={`text-2xl font-bold ${textColor}`}>Utilities</h2>
          <p className={textSecondary}>🏠 {selectedProperty}</p>
        </div>
        <button className="flex items-center gap-2 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600">
          <Plus size={20} />
          Aggiungi
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {mockUtilities.map(utility => {
          const Icon = utility.icon;
          const colorClasses = {
            yellow: 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800',
            orange: 'bg-orange-50 dark:bg-orange-900/20 border-orange-200 dark:border-orange-800',
            cyan: 'bg-cyan-50 dark:bg-cyan-900/20 border-cyan-200 dark:border-cyan-800',
            green: 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800',
          };
          const iconColors = {
            yellow: 'text-yellow-500',
            orange: 'text-orange-500',
            cyan: 'text-cyan-500',
            green: 'text-green-500',
          };

          return (
            <div key={utility.id} className={`${cardBg} rounded-xl p-6 border ${borderColor} hover:shadow-lg transition-all cursor-pointer backdrop-blur-xl bg-opacity-70`}>
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className={`p-3 rounded-xl border ${colorClasses[utility.color]}`}>
                    <Icon className={iconColors[utility.color]} size={24} />
                  </div>
                  <div>
                    <h3 className={`font-bold ${textColor}`}>{utility.name}</h3>
                    <p className={`text-sm ${textSecondary}`}>{utility.provider}</p>
                  </div>
                </div>
              </div>

              <div className="space-y-2 mb-4">
                <div className="flex justify-between">
                  <span className={textSecondary}>Consumo:</span>
                  <span className={textColor}>{utility.consumption}</span>
                </div>
                <div className="flex justify-between">
                  <span className={textSecondary}>Ultima bolletta:</span>
                  <span className={`font-medium ${textColor}`}>€{utility.lastBill.amount.toFixed(2)}</span>
                </div>
                <div className="flex justify-between">
                  <span className={textSecondary}>Scadenza:</span>
                  <span className={textColor}>{utility.lastBill.date}</span>
                </div>
                {utility.lastReading && (
                  <div className="flex justify-between">
                    <span className={textSecondary}>Ultima lettura:</span>
                    <span className={textColor}>{utility.lastReading}</span>
                  </div>
                )}
              </div>

              {utility.alert && (
                <div className="mb-4 p-3 bg-orange-50 dark:bg-orange-900/20 rounded-lg text-sm text-orange-600 dark:text-orange-400 border border-orange-200 dark:border-orange-800">
                  ⚠️ {utility.alert}
                </div>
              )}

              <div className="flex gap-2">
                <button className={`flex-1 px-3 py-2 text-sm border ${borderColor} rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors`}>
                  📊 Dettagli
                </button>
                <button className={`flex-1 px-3 py-2 text-sm border ${borderColor} rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors`}>
                  📄 Bollette
                </button>
                <button className={`flex-1 px-3 py-2 text-sm border ${borderColor} rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors`}>
                  📸 Lettura
                </button>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );

  const ProjectsView = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className={`text-2xl font-bold ${textColor}`}>Progetti di Spesa</h2>
        <button className="flex items-center gap-2 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600">
          <Plus size={20} />
          Nuovo Progetto
        </button>
      </div>

      <div className="space-y-4">
        {mockProjects.map(project => (
          <div key={project.id} className={`${cardBg} rounded-xl p-6 border ${borderColor} hover:shadow-lg transition-shadow cursor-pointer`}>
            <div className="flex items-start justify-between mb-4">
              <div>
                <h3 className={`text-lg font-bold ${textColor} mb-1`}>{project.name}</h3>
                <p className={textSecondary}>
                  📅 {project.startDate} - {project.endDate}
                  {project.property && <span> • 🏠 {project.property}</span>}
                </p>
              </div>
            </div>

            <div className="mb-4">
              <div className="flex justify-between mb-2">
                <span className={textSecondary}>Budget: €{project.budget.toFixed(2)}</span>
                <span className={`font-medium ${textColor}`}>
                  Speso: €{project.spent.toFixed(2)} ({((project.spent / project.budget) * 100).toFixed(0)}%)
                </span>
              </div>
              <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
                <div 
                  className="bg-blue-500 h-3 rounded-full transition-all"
                  style={{ width: `${(project.spent / project.budget) * 100}%` }}
                />
              </div>
            </div>

            <div className="flex items-center justify-between">
              <span className={textSecondary}>{project.expenseCount} spese collegate</span>
              <button className={`px-4 py-2 text-sm border ${borderColor} rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700`}>
                Dettagli →
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );

  const SettingsView = () => (
    <div className="space-y-6">
      <h2 className={`text-2xl font-bold ${textColor}`}>Impostazioni</h2>

      <div className={`${cardBg} rounded-xl p-6 border ${borderColor}`}>
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
      </div>

      <div className={`${cardBg} rounded-xl p-6 border ${borderColor}`}>
        <h3 className={`text-lg font-semibold ${textColor} mb-4`}>🏠 Casa Predefinita</h3>
        <select className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
          <option>Padova - Via Roma 71/B</option>
          <option>Genova Pontedecimo</option>
        </select>
      </div>

      <div className={`${cardBg} rounded-xl p-6 border ${borderColor}`}>
        <h3 className={`text-lg font-semibold ${textColor} mb-4`}>🌍 Localizzazione</h3>
        <div className="space-y-3">
          <div>
            <label className={`block text-sm ${textSecondary} mb-1`}>Lingua</label>
            <select className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
              <option>Italiano</option>
              <option>English</option>
            </select>
          </div>
          <div>
            <label className={`block text-sm ${textSecondary} mb-1`}>Valuta</label>
            <select className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
              <option>EUR (€)</option>
              <option>USD ($)</option>
            </select>
          </div>
        </div>
      </div>

      <div className={`${cardBg} rounded-xl p-6 border ${borderColor}`}>
        <h3 className={`text-lg font-semibold ${textColor} mb-4`}>🔔 Notifiche</h3>
        <div className="space-y-3">
          <label className="flex items-center justify-between cursor-pointer">
            <span className={textColor}>Email</span>
            <input type="checkbox" className="w-5 h-5" defaultChecked />
          </label>
          <label className="flex items-center justify-between cursor-pointer">
            <span className={textColor}>In-app</span>
            <input type="checkbox" className="w-5 h-5" defaultChecked />
          </label>
          <label className="flex items-center justify-between cursor-pointer">
            <span className={textColor}>Scadenze bollette</span>
            <input type="checkbox" className="w-5 h-5" defaultChecked />
          </label>
          <label className="flex items-center justify-between cursor-pointer">
            <span className={textColor}>Promemoria autoletture</span>
            <input type="checkbox" className="w-5 h-5" defaultChecked />
          </label>
        </div>
      </div>

      <button className="w-full px-4 py-3 bg-red-500 text-white rounded-lg hover:bg-red-600 font-medium">
        🚪 Logout
      </button>
    </div>
  );

  const AddExpenseModal = () => (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={() => setShowAddExpense(false)}>
      <div className={`${cardBg} rounded-2xl p-6 max-w-md w-full max-h-[90vh] overflow-y-auto`} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-6">
          <h3 className={`text-xl font-bold ${textColor}`}>Nuova Spesa</h3>
          <button onClick={() => setShowAddExpense(false)}>
            <X className={textColor} />
          </button>
        </div>

        <div className="mb-4 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
          <p className={`text-sm ${textColor}`}>
            🏠 Casa corrente: <strong>{selectedProperty}</strong>
          </p>
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
            <label className={`block text-sm ${textSecondary} mb-1`}>Categoria *</label>
            <select className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
              <option>🍕 Alimentari e Ristorazione</option>
              <option>🏠 Casa</option>
              <option>🚗 Trasporti</option>
              <option>🎬 Intrattenimento</option>
            </select>
          </div>

          <div>
            <label className={`block text-sm ${textSecondary} mb-1`}>Sottocategoria</label>
            <select className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
              <option>Ristoranti/Pizzerie</option>
              <option>Spesa Supermercato</option>
              <option>Bar/Caffè</option>
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

          <div>
            <label className={`block text-sm ${textSecondary} mb-1`}>Progetto (opzionale)</label>
            <select className={`w-full px-3 py-2 border ${borderColor} rounded-lg ${cardBg} ${textColor}`}>
              <option>Nessuno</option>
              <option>🏠 Ristrutturazione Bagno 2025</option>
              <option>✈️ Viaggio Giappone 2025</option>
            </select>
          </div>

          <div>
            <label className={`block text-sm ${textSecondary} mb-2`}>📎 Allega Scontrino/Fattura</label>
            <div className={`border-2 border-dashed ${borderColor} rounded-lg p-8 text-center cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50`}>
              <FileText className={`mx-auto mb-2 ${textSecondary}`} size={32} />
              <p className={textSecondary}>Clicca o trascina qui</p>
            </div>
          </div>
        </div>

        <div className="flex gap-3 mt-6">
          <button 
            onClick={() => setShowAddExpense(false)}
            className={`flex-1 px-4 py-3 border ${borderColor} rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 ${textColor}`}
          >
            Annulla
          </button>
          <button className="flex-1 px-4 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600">
            Salva
          </button>
        </div>
      </div>
    </div>
  );

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
          {currentView === 'expenses' && <ExpensesView />}
          {currentView === 'utilities' && <UtilitiesView />}
          {currentView === 'projects' && <ProjectsView />}
          {currentView === 'settings' && <SettingsView />}
        </main>
      </div>

      <MobileNav />
      <FAB />
      {showAddExpense && <AddExpenseModal />}
    </div>
  );
};

export default HomeLogPrototype;
