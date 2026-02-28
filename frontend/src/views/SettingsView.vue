<template>
  <div class="space-y-6">
    <!-- Header -->
    <div>
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Impostazioni</h1>
      <p class="text-gray-600 dark:text-gray-400 mt-1">Configura le preferenze dell'app</p>
    </div>

    <!-- Household Settings -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Impostazioni Famiglia</h2>

      <div class="space-y-4">
        <!-- Split Mode Toggle -->
        <div class="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-xl">
          <div class="flex-1">
            <div class="font-medium text-gray-900 dark:text-white">Modalita Split</div>
            <div class="text-sm text-gray-600 dark:text-gray-400 mt-1">
              Traccia chi deve cosa dividendo le spese tra i membri della famiglia
            </div>
          </div>
          <label class="relative inline-flex items-center cursor-pointer ml-4">
            <input
              type="checkbox"
              v-model="splitMode"
              @change="updateSplitMode"
              class="sr-only peer"
            />
            <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4
                        peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer
                        dark:bg-gray-600 peer-checked:after:translate-x-full
                        peer-checked:after:border-white after:content-[''] after:absolute
                        after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300
                        after:border after:rounded-full after:h-5 after:w-5 after:transition-all
                        dark:border-gray-500 peer-checked:bg-blue-600">
            </div>
          </label>
        </div>

        <!-- Split Settings (only if Split Mode ON) -->
        <div v-if="splitMode" class="pl-6 space-y-4 border-l-2 border-blue-200 dark:border-blue-800">
          <!-- Default split with users -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
              Dividi automaticamente le spese con:
            </label>

            <!-- User list -->
            <div v-if="householdMembers.length > 0" class="space-y-2">
              <div
                v-for="member in householdMembers"
                :key="member.id"
                class="flex items-center gap-3 p-3 bg-white dark:bg-gray-800 rounded-lg
                       border border-gray-200 dark:border-gray-600 hover:bg-gray-50
                       dark:hover:bg-gray-700 transition-colors"
              >
                <input
                  type="checkbox"
                  :value="member.id"
                  v-model="defaultSplitMemberIds"
                  @change="updateUserSettings"
                  class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500 cursor-pointer"
                />
                <div class="flex items-center gap-2 flex-1">
                  <div class="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900
                              flex items-center justify-center text-sm font-medium
                              text-blue-600 dark:text-blue-300">
                    {{ getInitials(member.name) }}
                  </div>
                  <span class="text-gray-900 dark:text-white">{{ member.name }}</span>
                  <span v-if="member.is_virtual" class="text-xs text-gray-500 dark:text-gray-400">(virtuale)</span>
                </div>
                <button
                  v-if="member.is_virtual"
                  @click="deleteMember(member.id)"
                  class="text-red-500 hover:text-red-700 dark:hover:text-red-400 p-1"
                  title="Elimina membro"
                >
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            </div>

            <!-- No members message -->
            <div v-if="householdMembers.length === 0" class="text-sm text-gray-600 dark:text-gray-400 italic p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
              Nessun altro membro nella casa. Aggiungi un membro per dividere le spese.
            </div>

            <!-- Add new member -->
            <div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
              <div class="flex gap-2">
                <input
                  v-model="newMemberName"
                  type="text"
                  placeholder="Nome nuovo membro (es: Partner)"
                  class="flex-1 px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                         bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                         focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                  @keyup.enter="addMember"
                />
                <Button @click="addMember" :disabled="!newMemberName.trim()">
                  Aggiungi
                </Button>
              </div>
            </div>

            <!-- Hint -->
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
              Quando aggiungi una nuova spesa, sara automaticamente divisa con le persone selezionate.
              Puoi sempre modificare per singola spesa.
            </p>
          </div>
        </div>

        <!-- Info Split Mode -->
        <div v-if="splitMode" class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
          <div class="text-sm text-gray-700 dark:text-gray-300">
            <div class="font-medium mb-2">Split Mode Attivo</div>
            <ul class="list-disc list-inside space-y-1 text-gray-600 dark:text-gray-400">
              <li>Ogni spesa puo essere divisa tra membri della famiglia</li>
              <li>Il sistema traccia chi deve cosa</li>
              <li>Puoi saldare i conti dalla pagina Bilancio</li>
            </ul>
          </div>
        </div>
      </div>
    </Card>

    <!-- Property Management (Admin only) -->
    <Card v-if="isAdmin" class="p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h2 class="text-xl font-bold text-gray-900 dark:text-white">Gestione Abitazioni</h2>
          <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">Aggiungi una nuova abitazione al sistema</p>
        </div>
        <Button @click="showAddPropertyForm = !showAddPropertyForm">
          {{ showAddPropertyForm ? 'Annulla' : '+ Nuova Abitazione' }}
        </Button>
      </div>

      <!-- Existing properties list -->
      <div v-if="allProperties.length > 0" class="space-y-2 mb-4">
        <div
          v-for="prop in allProperties"
          :key="prop.id"
          class="flex items-center gap-3 p-3 rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800"
        >
          <span class="text-2xl">🏠</span>
          <div class="flex-1 min-w-0">
            <div class="font-medium text-gray-900 dark:text-white">{{ prop.name }}</div>
            <div v-if="prop.address" class="text-xs text-gray-500 dark:text-gray-400 truncate">{{ prop.address }}</div>
          </div>
          <span v-if="prop.is_current" class="px-2 py-0.5 text-xs rounded-full bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300 font-medium">Principale</span>
        </div>
      </div>

      <!-- Add property form -->
      <div v-if="showAddPropertyForm" class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-xl border border-blue-200 dark:border-blue-800 space-y-3">
        <h3 class="text-sm font-medium text-gray-900 dark:text-white">Nuova Abitazione</h3>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <input
            v-model="newProperty.name"
            type="text"
            placeholder="Nome abitazione *"
            class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
          />
          <input
            v-model="newProperty.address"
            type="text"
            placeholder="Indirizzo"
            class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
          />
          <select
            v-model="newProperty.type"
            class="px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
          >
            <option value="owned">Di proprietà</option>
            <option value="rented">In affitto</option>
          </select>
          <div class="flex items-center gap-2">
            <input
              type="checkbox"
              id="prop-is-current"
              v-model="newProperty.is_current"
              class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
            />
            <label for="prop-is-current" class="text-sm text-gray-700 dark:text-gray-300 cursor-pointer">
              Imposta come principale
            </label>
          </div>
        </div>
        <div v-if="propertyError" class="text-sm text-red-600 dark:text-red-400">{{ propertyError }}</div>
        <Button @click="addProperty" :disabled="!newProperty.name.trim() || propertyLoading">
          {{ propertyLoading ? 'Creazione...' : 'Crea Abitazione' }}
        </Button>
      </div>
    </Card>

    <!-- User Preferences -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Preferenze Personali</h2>

      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Tema</label>
          <select
            v-model="preferences.theme"
            @change="updateUserSettings"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="light">Chiaro</option>
            <option value="dark">Scuro</option>
            <option value="auto">Automatico</option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Valuta</label>
          <select
            v-model="preferences.currency"
            @change="updateUserSettings"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="EUR">EUR (Euro)</option>
            <option value="USD">USD (Dollaro)</option>
            <option value="GBP">GBP (Sterlina)</option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Lingua</label>
          <select
            v-model="preferences.language"
            @change="updateUserSettings"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="it">Italiano</option>
            <option value="en">English</option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Formato Data</label>
          <select
            v-model="preferences.date_format"
            @change="updateUserSettings"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="DD/MM/YYYY">GG/MM/AAAA (31/12/2024)</option>
            <option value="MM/DD/YYYY">MM/GG/AAAA (12/31/2024)</option>
            <option value="YYYY-MM-DD">AAAA-MM-GG (2024-12-31)</option>
            <option value="DD MMM YYYY">GG MMM AAAA (31 dic 2024)</option>
          </select>
        </div>
      </div>
    </Card>

    <!-- Default Templates -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Template Predefiniti Bollette</h2>
      <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
        Seleziona il template predefinito per ogni tipo di utenza. Verra usato automaticamente quando carichi una nuova bolletta.
      </p>

      <div class="space-y-4">
        <div v-for="utilityType in utilityTypes" :key="utilityType.key">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ utilityType.label }}
          </label>
          <select
            v-model="defaultTemplates[utilityType.key]"
            @change="updateUserSettings"
            class="w-full px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option :value="null">Nessun template (rilevamento automatico)</option>
            <option
              v-for="tpl in getTemplatesForType(utilityType.key)"
              :key="tpl.id"
              :value="tpl.id"
            >
              {{ tpl.name }} - {{ tpl.provider }}
            </option>
          </select>
        </div>

        <div v-if="allTemplates.length === 0" class="text-sm text-gray-500 dark:text-gray-400 italic p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
          Nessun template disponibile. Crea i template dalla pagina Utenze.
        </div>
      </div>
    </Card>

    <!-- Category Management -->
    <Card class="p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h2 class="text-xl font-bold text-gray-900 dark:text-white">Gestione Categorie</h2>
          <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Gestisci le categorie per organizzare le spese
          </p>
        </div>
        <Button @click="showAddCategoryForm = true" v-if="!showAddCategoryForm">
          + Nuova Categoria
        </Button>
      </div>

      <!-- Add Category Form -->
      <div v-if="showAddCategoryForm" class="mb-4 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-xl border border-blue-200 dark:border-blue-800">
        <h3 class="text-sm font-medium text-gray-900 dark:text-white mb-3">Nuova categoria personale</h3>
        <div class="flex gap-2 mb-2">
          <input
            v-model="newCategory.icon"
            type="text"
            placeholder="🏠"
            maxlength="4"
            class="w-16 px-2 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-center
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <input
            v-model="newCategory.name"
            type="text"
            placeholder="Nome categoria"
            class="flex-1 px-3 py-2 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
            @keyup.enter="addCategory"
          />
        </div>
        <div v-if="isAdmin" class="flex items-center gap-2 mb-3">
          <input
            type="checkbox"
            id="cat-is-default"
            v-model="newCategory.is_default"
            class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
          />
          <label for="cat-is-default" class="text-sm text-gray-700 dark:text-gray-300 cursor-pointer">
            Categoria predefinita (visibile a tutti)
          </label>
        </div>
        <div class="flex gap-2">
          <Button @click="addCategory" :disabled="!newCategory.name.trim()">Salva</Button>
          <Button variant="secondary" @click="showAddCategoryForm = false; newCategory = { name: '', icon: '', is_default: false }">Annulla</Button>
        </div>
      </div>

      <!-- Category List -->
      <div v-if="categories.length === 0 && !categoriesLoading" class="text-sm text-gray-500 dark:text-gray-400 italic p-4 bg-gray-50 dark:bg-gray-700 rounded-lg text-center">
        Nessuna categoria disponibile.
      </div>

      <div class="space-y-2">
        <!-- Default categories section -->
        <div v-if="defaultCategories.length > 0">
          <div class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-2">
            Categorie predefinite
          </div>
          <div
            v-for="cat in defaultCategories"
            :key="cat.id"
            class="border border-gray-200 dark:border-gray-700 rounded-xl overflow-hidden"
          >
            <!-- Category header -->
            <div
              class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 cursor-pointer"
              @click="toggleCategory(cat.id)"
            >
              <span class="text-lg w-7 text-center">{{ cat.icon }}</span>
              <span class="flex-1 font-medium text-gray-900 dark:text-white">{{ cat.name }}</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                {{ cat.subcategories?.length || 0 }} sottocategorie
              </span>
              <div class="flex items-center gap-1">
                <button
                  v-if="isAdmin"
                  @click.stop="startAddSubcategory(cat)"
                  class="p-1 text-blue-500 hover:text-blue-700 dark:hover:text-blue-400"
                  title="Aggiungi sottocategoria"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                  </svg>
                </button>
                <button
                  v-if="isAdmin"
                  @click.stop="deleteCategory(cat)"
                  class="p-1 text-red-400 hover:text-red-600 dark:hover:text-red-400"
                  title="Elimina categoria"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
                <svg
                  :class="['w-4 h-4 text-gray-400 transition-transform', expandedCategories.has(cat.id) ? 'rotate-180' : '']"
                  fill="none" stroke="currentColor" viewBox="0 0 24 24"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </div>

            <!-- Subcategories -->
            <div v-if="expandedCategories.has(cat.id)" class="border-t border-gray-200 dark:border-gray-700">
              <!-- Add subcategory inline form -->
              <div v-if="addSubcategoryForCat === cat.id" class="p-3 bg-blue-50 dark:bg-blue-900/20 border-b border-blue-200 dark:border-blue-800">
                <div class="flex gap-2">
                  <input
                    v-model="newSubcategoryName"
                    type="text"
                    placeholder="Nome sottocategoria"
                    class="flex-1 px-2 py-1 text-sm border border-gray-200 dark:border-gray-700 rounded
                           bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                           focus:outline-none focus:ring-1 focus:ring-blue-500"
                    @keyup.enter="saveSubcategory(cat.id)"
                    ref="subcategoryInput"
                  />
                  <button @click="saveSubcategory(cat.id)" class="px-3 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700">Aggiungi</button>
                  <button @click="addSubcategoryForCat = null; newSubcategoryName = ''" class="px-3 py-1 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800">Annulla</button>
                </div>
              </div>

              <div v-if="!cat.subcategories?.length" class="px-4 py-2 text-sm text-gray-400 italic">
                Nessuna sottocategoria
              </div>
              <div
                v-for="sub in cat.subcategories"
                :key="sub.id"
                class="flex items-center gap-2 px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-700/50"
              >
                <span class="w-4 h-4 text-gray-400">·</span>
                <span class="flex-1 text-sm text-gray-700 dark:text-gray-300">{{ sub.name }}</span>
                <button
                  v-if="isAdmin"
                  @click="deleteSubcategory(cat, sub)"
                  class="p-1 text-red-400 hover:text-red-600 opacity-60 hover:opacity-100"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Personal categories section -->
        <div v-if="personalCategories.length > 0" :class="defaultCategories.length > 0 ? 'mt-4' : ''">
          <div class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-2">
            Le mie categorie
          </div>
          <div
            v-for="cat in personalCategories"
            :key="cat.id"
            class="border border-gray-200 dark:border-gray-700 rounded-xl overflow-hidden"
          >
            <div
              class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 cursor-pointer"
              @click="toggleCategory(cat.id)"
            >
              <span class="text-lg w-7 text-center">{{ cat.icon }}</span>
              <span class="flex-1 font-medium text-gray-900 dark:text-white">{{ cat.name }}</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                {{ cat.subcategories?.length || 0 }} sottocategorie
              </span>
              <div class="flex items-center gap-1">
                <button
                  @click.stop="startAddSubcategory(cat)"
                  class="p-1 text-blue-500 hover:text-blue-700 dark:hover:text-blue-400"
                  title="Aggiungi sottocategoria"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                  </svg>
                </button>
                <button
                  @click.stop="deleteCategory(cat)"
                  class="p-1 text-red-400 hover:text-red-600 dark:hover:text-red-400"
                  title="Elimina categoria"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
                <svg
                  :class="['w-4 h-4 text-gray-400 transition-transform', expandedCategories.has(cat.id) ? 'rotate-180' : '']"
                  fill="none" stroke="currentColor" viewBox="0 0 24 24"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </div>

            <div v-if="expandedCategories.has(cat.id)" class="border-t border-gray-200 dark:border-gray-700">
              <div v-if="addSubcategoryForCat === cat.id" class="p-3 bg-blue-50 dark:bg-blue-900/20 border-b border-blue-200 dark:border-blue-800">
                <div class="flex gap-2">
                  <input
                    v-model="newSubcategoryName"
                    type="text"
                    placeholder="Nome sottocategoria"
                    class="flex-1 px-2 py-1 text-sm border border-gray-200 dark:border-gray-700 rounded
                           bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                           focus:outline-none focus:ring-1 focus:ring-blue-500"
                    @keyup.enter="saveSubcategory(cat.id)"
                  />
                  <button @click="saveSubcategory(cat.id)" class="px-3 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700">Aggiungi</button>
                  <button @click="addSubcategoryForCat = null; newSubcategoryName = ''" class="px-3 py-1 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800">Annulla</button>
                </div>
              </div>

              <div v-if="!cat.subcategories?.length" class="px-4 py-2 text-sm text-gray-400 italic">
                Nessuna sottocategoria
              </div>
              <div
                v-for="sub in cat.subcategories"
                :key="sub.id"
                class="flex items-center gap-2 px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-700/50"
              >
                <span class="w-4 h-4 text-gray-400">·</span>
                <span class="flex-1 text-sm text-gray-700 dark:text-gray-300">{{ sub.name }}</span>
                <button
                  @click="deleteSubcategory(cat, sub)"
                  class="p-1 text-red-400 hover:text-red-600 opacity-60 hover:opacity-100"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="categoryError" class="mt-3 text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
        {{ categoryError }}
      </div>
    </Card>

    <!-- Account Info -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Account</h2>

      <div class="flex items-center gap-4">
        <div class="w-16 h-16 rounded-full bg-gradient-to-br from-blue-500 to-purple-600
                    flex items-center justify-center text-white text-xl font-bold">
          {{ userInitials }}
        </div>
        <div>
          <div class="font-medium text-gray-900 dark:text-white">{{ authStore.user?.name }}</div>
          <div class="text-sm text-gray-600 dark:text-gray-400">{{ authStore.user?.email }}</div>
          <div class="text-xs text-gray-500 dark:text-gray-500 mt-1">
            {{ authStore.user?.role === 'admin' ? 'Amministratore' : 'Utente' }}
          </div>
        </div>
      </div>

      <div class="mt-6 pt-6 border-t border-gray-200 dark:border-gray-700">
        <Button variant="danger" @click="handleLogout">
          Esci dall'account
        </Button>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { templatesAPI, categoriesAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import apiClient from '@/api/client'

const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

const splitMode = ref(false)
const currentPropertyId = ref(null)
const householdMembers = ref([])
const defaultSplitMemberIds = ref([])
const currentUserMemberId = ref(null)
const newMemberName = ref('')

// Templates
const allTemplates = ref([])
const defaultTemplates = ref({
  electricity: null,
  gas: null,
  water: null,
  waste: null
})

const utilityTypes = [
  { key: 'electricity', label: 'Luce' },
  { key: 'gas', label: 'Gas' },
  { key: 'water', label: 'Acqua' },
  { key: 'waste', label: 'Rifiuti' }
]

function getTemplatesForType(type) {
  return allTemplates.value.filter(t => t.utility_type === type)
}

const preferences = ref({
  theme: 'auto',
  currency: 'EUR',
  language: 'it',
  date_format: 'DD/MM/YYYY'
})

// Admin check
const isAdmin = computed(() => authStore.user?.role === 'admin')

// Property management (admin)
const allProperties = ref([])
const showAddPropertyForm = ref(false)
const propertyLoading = ref(false)
const propertyError = ref(null)
const newProperty = ref({ name: '', address: '', type: 'owned', is_current: false })

async function fetchAllProperties() {
  try {
    const { data } = await apiClient.get('/properties')
    allProperties.value = data || []
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

async function addProperty() {
  if (!newProperty.value.name.trim()) return
  propertyLoading.value = true
  propertyError.value = null
  try {
    await apiClient.post('/properties', {
      name: newProperty.value.name.trim(),
      address: newProperty.value.address.trim(),
      type: newProperty.value.type,
      is_current: newProperty.value.is_current
    })
    newProperty.value = { name: '', address: '', type: 'owned', is_current: false }
    showAddPropertyForm.value = false
    await fetchAllProperties()
    // Re-fetch current property in case it changed
    await fetchCurrentProperty()
  } catch (err) {
    propertyError.value = err.response?.data?.error || 'Errore durante la creazione'
  } finally {
    propertyLoading.value = false
  }
}

// Category management
const categories = ref([])
const categoriesLoading = ref(false)
const categoryError = ref(null)
const expandedCategories = ref(new Set())
const showAddCategoryForm = ref(false)
const newCategory = ref({ name: '', icon: '', is_default: false })
const addSubcategoryForCat = ref(null)
const newSubcategoryName = ref('')

const defaultCategories = computed(() => categories.value.filter(c => c.is_default))
const personalCategories = computed(() => categories.value.filter(c => !c.is_default))

function toggleCategory(id) {
  if (expandedCategories.value.has(id)) {
    expandedCategories.value.delete(id)
  } else {
    expandedCategories.value.add(id)
  }
  // trigger reactivity
  expandedCategories.value = new Set(expandedCategories.value)
}

async function fetchCategories() {
  categoriesLoading.value = true
  try {
    const { data } = await categoriesAPI.list()
    categories.value = data || []
  } catch (err) {
    console.error('Error fetching categories:', err)
  } finally {
    categoriesLoading.value = false
  }
}

async function addCategory() {
  if (!newCategory.value.name.trim()) return
  categoryError.value = null
  try {
    await categoriesAPI.create({
      name: newCategory.value.name.trim(),
      icon: newCategory.value.icon.trim() || '📁',
      is_default: isAdmin.value && newCategory.value.is_default
    })
    newCategory.value = { name: '', icon: '', is_default: false }
    showAddCategoryForm.value = false
    await fetchCategories()
  } catch (err) {
    categoryError.value = err.response?.data?.error || 'Errore durante la creazione'
  }
}

async function deleteCategory(cat) {
  const label = cat.is_default ? 'categoria predefinita' : 'categoria personale'
  if (!confirm(`Eliminare la ${label} "${cat.name}"? Anche tutte le sue sottocategorie verranno eliminate.`)) return
  categoryError.value = null
  try {
    await categoriesAPI.delete(cat.id)
    await fetchCategories()
  } catch (err) {
    categoryError.value = err.response?.data?.error || 'Errore durante l\'eliminazione'
  }
}

function startAddSubcategory(cat) {
  addSubcategoryForCat.value = cat.id
  newSubcategoryName.value = ''
  // Make sure category is expanded
  if (!expandedCategories.value.has(cat.id)) {
    expandedCategories.value.add(cat.id)
    expandedCategories.value = new Set(expandedCategories.value)
  }
}

async function saveSubcategory(catId) {
  if (!newSubcategoryName.value.trim()) return
  categoryError.value = null
  try {
    await categoriesAPI.createSubcategory(catId, { name: newSubcategoryName.value.trim() })
    addSubcategoryForCat.value = null
    newSubcategoryName.value = ''
    await fetchCategories()
  } catch (err) {
    categoryError.value = err.response?.data?.error || 'Errore durante la creazione'
  }
}

async function deleteSubcategory(cat, sub) {
  if (!confirm(`Eliminare la sottocategoria "${sub.name}"?`)) return
  categoryError.value = null
  try {
    await categoriesAPI.deleteSubcategory(cat.id, sub.id)
    await fetchCategories()
  } catch (err) {
    categoryError.value = err.response?.data?.error || 'Errore durante l\'eliminazione'
  }
}

const userInitials = computed(() => {
  const name = authStore.user?.name || 'U'
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
})

function getInitials(name) {
  return name
    .split(' ')
    .map(n => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

async function fetchCurrentProperty() {
  try {
    const { data } = await apiClient.get('/properties')
    if (data && data.length > 0) {
      const currentProp = data.find(p => p.is_current) || data[0]
      currentPropertyId.value = currentProp.id
      console.log('Current property ID:', currentProp.id)
      await loadHouseholdSettings()
      await fetchHouseholdMembers()
    }
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

async function fetchHouseholdMembers() {
  if (!currentPropertyId.value) return

  try {
    const { data } = await apiClient.get(`/properties/${currentPropertyId.value}/members`)
    // Filter out current user's member from the list (only show other members)
    const currentUserId = authStore.user?.id
    householdMembers.value = data.filter(m => m.user_id !== currentUserId)

    // Find current user's member ID
    const myMember = data.find(m => m.user_id === currentUserId)
    if (myMember) {
      currentUserMemberId.value = myMember.id
    }

    console.log('Household members:', householdMembers.value)
    console.log('Current user member ID:', currentUserMemberId.value)
  } catch (err) {
    console.log('Using empty members list')
    householdMembers.value = []
  }
}

async function loadHouseholdSettings() {
  if (!currentPropertyId.value) return

  try {
    const { data } = await apiClient.get(`/properties/${currentPropertyId.value}/settings`)
    splitMode.value = data.split_mode || false
  } catch (err) {
    console.log('Using default household settings')
    splitMode.value = false
  }
}

async function loadUserSettings() {
  try {
    const { data } = await apiClient.get('/settings')
    preferences.value = {
      theme: data.theme || 'auto',
      currency: data.currency || 'EUR',
      language: data.language || 'it',
      date_format: data.date_format || 'DD/MM/YYYY'
    }

    // Parse default split member IDs
    if (data.default_split_with_member_ids) {
      try {
        defaultSplitMemberIds.value = JSON.parse(data.default_split_with_member_ids)
      } catch (e) {
        defaultSplitMemberIds.value = []
      }
    }

    // Parse default templates
    if (data.default_templates) {
      try {
        const parsed = JSON.parse(data.default_templates)
        defaultTemplates.value = {
          electricity: parsed.electricity || null,
          gas: parsed.gas || null,
          water: parsed.water || null,
          waste: parsed.waste || null
        }
      } catch (e) {
        console.error('Error parsing default_templates:', e)
      }
    }
  } catch (err) {
    console.log('Using default user settings')
  }
}

async function loadTemplates() {
  try {
    const { data } = await templatesAPI.listBillTemplates()
    allTemplates.value = data || []
  } catch (err) {
    console.error('Error loading templates:', err)
    allTemplates.value = []
  }
}

async function updateSplitMode() {
  if (!currentPropertyId.value) {
    console.error('No property selected')
    return
  }

  try {
    await apiClient.put(`/properties/${currentPropertyId.value}/settings`, {
      split_mode: splitMode.value
    })
    console.log('Split mode updated:', splitMode.value)
  } catch (err) {
    console.error('Error updating split mode:', err)
    splitMode.value = !splitMode.value
  }
}

async function updateUserSettings() {
  try {
    const payload = {
      theme: preferences.value.theme,
      currency: preferences.value.currency,
      language: preferences.value.language,
      date_format: preferences.value.date_format,
      default_split_with_member_ids: JSON.stringify(defaultSplitMemberIds.value),
      default_templates: JSON.stringify(defaultTemplates.value)
    }
    await apiClient.put('/settings', payload)
    // Sync with global settings store
    settingsStore.theme = preferences.value.theme
    settingsStore.currency = preferences.value.currency
    settingsStore.language = preferences.value.language
    settingsStore.dateFormat = preferences.value.date_format
    console.log('User settings updated:', payload)
  } catch (err) {
    console.error('Error updating user settings:', err)
  }
}

async function addMember() {
  if (!newMemberName.value.trim() || !currentPropertyId.value) return

  try {
    await apiClient.post(`/properties/${currentPropertyId.value}/members`, {
      name: newMemberName.value.trim(),
      role: ''
    })
    newMemberName.value = ''
    await fetchHouseholdMembers()
    console.log('Member added successfully')
  } catch (err) {
    console.error('Error adding member:', err)
  }
}

async function deleteMember(memberId) {
  if (!confirm('Sei sicuro di voler eliminare questo membro?')) return

  try {
    await apiClient.delete(`/members/${memberId}`)
    // Remove from default split if present
    defaultSplitMemberIds.value = defaultSplitMemberIds.value.filter(id => id !== memberId)
    await updateUserSettings()
    await fetchHouseholdMembers()
    console.log('Member deleted successfully')
  } catch (err) {
    console.error('Error deleting member:', err)
    if (err.response?.data?.error) {
      alert(err.response.data.error)
    }
  }
}

function handleLogout() {
  authStore.logout()
  router.push('/login')
}

onMounted(() => {
  loadUserSettings()
  loadTemplates()
  fetchCurrentProperty()
  fetchCategories()
  if (authStore.user?.role === 'admin') {
    fetchAllProperties()
  }
})
</script>
