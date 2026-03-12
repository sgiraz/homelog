<template>
  <div class="space-y-4">
    <!-- Profile Card (Apple HIG: always visible above tabs) -->
    <Card class="p-4 sm:p-6">
      <div class="flex items-center gap-4">
        <!-- Avatar with always-visible camera badge -->
        <div class="relative shrink-0">
          <button
            @click="$refs.avatarInput.click()"
            class="block relative cursor-pointer"
            aria-label="Cambia foto profilo"
          >
            <img
              v-if="authStore.avatarUrl"
              :src="authStore.avatarUrl"
              :alt="authStore.user?.name"
              class="w-16 h-16 sm:w-20 sm:h-20 rounded-full object-cover"
            />
            <div
              v-else
              class="w-16 h-16 sm:w-20 sm:h-20 rounded-full bg-gradient-to-br from-blue-500 to-purple-600
                      flex items-center justify-center text-white text-xl sm:text-2xl font-bold"
            >
              {{ userInitials }}
            </div>
            <!-- Camera badge (always visible, Apple-style) -->
            <div
              class="absolute -bottom-0.5 -right-0.5 w-7 h-7 sm:w-8 sm:h-8
                     bg-gray-100 dark:bg-gray-600 rounded-full
                     flex items-center justify-center
                     border-2 border-white dark:border-gray-800
                     shadow-sm"
              :class="avatarUploading ? 'animate-pulse' : ''"
            >
              <svg v-if="!avatarUploading" class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-gray-600 dark:text-gray-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              <div v-else class="w-3.5 h-3.5 border-2 border-gray-600 dark:border-gray-200 border-t-transparent rounded-full animate-spin" />
            </div>
          </button>
          <input
            ref="avatarInput"
            type="file"
            accept="image/jpeg,image/png,image/webp"
            class="hidden"
            @change="onAvatarSelected"
          />
        </div>
        <div class="flex-1 min-w-0">
          <div class="font-semibold text-lg text-gray-900 dark:text-white truncate">{{ authStore.user?.name }}</div>
          <div class="text-sm text-gray-500 dark:text-gray-400 truncate">{{ authStore.user?.email }}</div>
          <div class="flex items-center gap-3 mt-1.5">
            <span class="text-xs text-gray-400 dark:text-gray-500">
              {{ authStore.user?.role === 'admin' ? 'Amministratore' : 'Utente' }}
            </span>
            <button
              v-if="authStore.avatarUrl"
              @click="removeAvatar"
              class="text-xs text-red-500 hover:text-red-700 dark:hover:text-red-400"
            >
              Rimuovi foto
            </button>
          </div>
        </div>
      </div>
    </Card>

    <!-- Tab Bar -->
    <div class="overflow-x-auto -mx-4 sm:mx-0 px-4 sm:px-0 pb-1">
      <div class="flex gap-1 min-w-max sm:min-w-0 sm:flex-wrap">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          @click="activeTab = tab.id"
          :class="[
            'flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium whitespace-nowrap transition-colors',
            activeTab === tab.id
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
          ]"
        >
          <span>{{ tab.icon }}</span>
          <span>{{ tab.label }}</span>
        </button>
      </div>
    </div>

    <!-- Tab: Famiglia -->
    <div v-show="activeTab === 'famiglia'" class="space-y-4">
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
            <label v-if="isAdmin" class="relative inline-flex items-center cursor-pointer ml-4">
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
            <span v-else class="ml-4 text-sm font-medium" :class="splitMode ? 'text-green-600 dark:text-green-400' : 'text-gray-400'">
              {{ splitMode ? 'Attivo' : 'Disattivo' }}
            </span>
          </div>

          <!-- Split Settings -->
          <div v-if="splitMode" class="pl-6 space-y-4 border-l-2 border-blue-200 dark:border-blue-800">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                Dividi automaticamente le spese con:
              </label>

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
                  <div class="flex items-center gap-2 flex-1 min-w-0">
                    <div class="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900
                                flex items-center justify-center text-sm font-medium
                                text-blue-600 dark:text-blue-300 flex-shrink-0">
                      {{ getInitials(member.name) }}
                    </div>
                    <span class="text-gray-900 dark:text-white truncate">{{ member.name }}</span>
                    <span v-if="member.is_virtual" class="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0">(virtuale)</span>
                    <span v-if="member.user_role === 'admin'" class="text-xs bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-300 px-1.5 py-0.5 rounded font-medium flex-shrink-0">Admin</span>
                  </div>
                  <div class="flex items-center gap-1 flex-shrink-0">
                    <!-- Admin: toggle admin role -->
                    <button
                      v-if="isAdmin && !member.is_virtual && member.user_id !== authStore.user?.id"
                      @click="toggleAdminRole(member)"
                      class="p-2 transition-colors"
                      :class="member.user_role === 'admin' ? 'text-amber-500 hover:text-amber-700' : 'text-gray-400 hover:text-amber-500'"
                      :title="member.user_role === 'admin' ? 'Rimuovi ruolo admin' : 'Promuovi ad admin'"
                    >
                      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                      </svg>
                    </button>
                    <!-- Admin: delete virtual member -->
                    <button
                      v-if="isAdmin && member.is_virtual"
                      @click="deleteMember(member.id)"
                      class="text-red-500 hover:text-red-700 dark:hover:text-red-400 p-2"
                      aria-label="Elimina membro"
                    >
                      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                    <!-- Admin: delete user account -->
                    <button
                      v-if="isAdmin && !member.is_virtual && member.user_id !== authStore.user?.id"
                      @click="deleteUserAccount(member)"
                      class="text-red-500 hover:text-red-700 dark:hover:text-red-400 p-2"
                      aria-label="Elimina account utente"
                      title="Elimina account e tutti i dati"
                    >
                      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728l-12.728-12.728" />
                      </svg>
                    </button>
                  </div>
                </div>
              </div>

              <div v-if="householdMembers.length === 0" class="text-sm text-gray-600 dark:text-gray-400 italic p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                Nessun altro membro nella casa. Aggiungi un membro per dividere le spese.
              </div>

              <!-- Add new member (admin only) -->
              <div v-if="isAdmin" class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
                <div class="flex flex-col sm:flex-row gap-2">
                  <input
                    v-model="newMemberName"
                    type="text"
                    placeholder="Nome membro"
                    class="flex-1 px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                           bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                           focus:outline-none focus:ring-2 focus:ring-blue-500"
                    @keyup.enter="addMember"
                  />
                  <Button @click="addMember" :disabled="!newMemberName.trim()">
                    Aggiungi
                  </Button>
                </div>
              </div>

              <p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
                Quando aggiungi una nuova spesa, sara automaticamente divisa con le persone selezionate.
              </p>
            </div>
          </div>

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
    </div>

    <!-- Tab: Proprietà -->
    <div v-show="activeTab === 'proprieta'" class="space-y-4">
      <Card v-if="isAdmin" class="p-6">
        <div class="flex items-center justify-between mb-4">
          <div>
            <h2 class="text-xl font-bold text-gray-900 dark:text-white">Gestione Abitazioni</h2>
            <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">Aggiungi una nuova abitazione al sistema</p>
          </div>
          <Button @click="showAddPropertyForm = !showAddPropertyForm" :variant="showAddPropertyForm ? 'secondary' : 'primary'" size="sm">
            <svg v-if="!showAddPropertyForm" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </Button>
        </div>

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

        <div v-if="showAddPropertyForm" class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-xl border border-blue-200 dark:border-blue-800 space-y-3">
          <h3 class="text-sm font-medium text-gray-900 dark:text-white">Nuova Abitazione</h3>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <input
              v-model="newProperty.name"
              type="text"
              placeholder="Nome abitazione *"
              class="px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              v-model="newProperty.address"
              type="text"
              placeholder="Indirizzo"
              autocomplete="street-address"
              class="px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <select
              v-model="newProperty.type"
              class="px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
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

      <Card v-if="!isAdmin" class="p-6">
        <p class="text-gray-600 dark:text-gray-400">Solo gli amministratori possono gestire le abitazioni.</p>
      </Card>
    </div>

    <!-- Tab: Preferenze -->
    <div v-show="activeTab === 'preferenze'" class="space-y-4">
      <Card class="p-6">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Preferenze Personali</h2>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Tema</label>
            <select
              v-model="preferences.theme"
              @change="updateUserSettings"
              class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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
              class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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
              class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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
              class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-2">Template Predefiniti Bollette</h2>
        <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
          Seleziona il template predefinito per ogni tipo di utenza.
        </p>

        <div class="space-y-4">
          <div v-for="utilityType in utilityTypes" :key="utilityType.key">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {{ utilityType.label }}
            </label>
            <select
              v-model="defaultTemplates[utilityType.key]"
              @change="updateUserSettings"
              class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option :value="null">Nessun template (auto)</option>
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

      <!-- Account -->
      <Card class="p-6">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Account</h2>
        <div class="space-y-3">
          <!-- Change Password toggle -->
          <div>
            <button
              type="button"
              @click="showChangePassword = !showChangePassword; pwError = null; pwSuccess = null"
              class="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 font-medium"
            >
              {{ showChangePassword ? '✕ Annulla' : 'Cambia password' }}
            </button>

            <div v-if="showChangePassword" class="mt-4 space-y-3">
              <Input
                v-model="pwForm.current"
                label="Password attuale"
                type="password"
                placeholder="Password attuale"
                autocomplete="current-password"
              />
              <Input
                v-model="pwForm.newPw"
                label="Nuova password"
                type="password"
                placeholder="Minimo 6 caratteri"
                autocomplete="new-password"
              />
              <Input
                v-model="pwForm.confirm"
                label="Conferma nuova password"
                type="password"
                placeholder="Ripeti la nuova password"
                autocomplete="new-password"
              />

              <div v-if="pwError" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
                {{ pwError }}
              </div>
              <div v-if="pwSuccess" class="text-green-700 text-sm bg-green-50 dark:bg-green-900/20 p-3 rounded-lg">
                {{ pwSuccess }}
              </div>

              <Button :disabled="pwLoading" @click="handleChangePassword">
                {{ pwLoading ? 'Salvataggio...' : 'Aggiorna password' }}
              </Button>
            </div>
          </div>

          <Button variant="danger" @click="handleLogout">
            Esci dall'account
          </Button>
        </div>
      </Card>
    </div>

    <!-- Tab: Categorie -->
    <div v-show="activeTab === 'categorie'" class="space-y-4">
      <Card class="p-6">
        <div class="flex items-center justify-between mb-4">
          <div>
            <h2 class="text-xl font-bold text-gray-900 dark:text-white">Gestione Categorie</h2>
            <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
              Gestisci le categorie per organizzare le spese
            </p>
          </div>
          <Button v-if="!showAddCategoryForm" @click="showAddCategoryForm = true" size="sm">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
          </Button>
        </div>

        <!-- Add Category Form -->
        <div v-if="showAddCategoryForm" class="mb-4 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-xl border border-blue-200 dark:border-blue-800 overflow-hidden">
          <h3 class="text-sm font-medium text-gray-900 dark:text-white mb-3">Nuova categoria personale</h3>
          <div class="flex gap-2 mb-2 min-w-0">
            <input
              v-model="newCategory.icon"
              type="text"
              placeholder="🏠"
              maxlength="4"
              class="w-14 shrink-0 px-2 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-center text-base
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              v-model="newCategory.name"
              type="text"
              placeholder="Nome categoria"
              class="flex-1 min-w-0 px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-base
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
          <!-- Default categories -->
          <div v-if="defaultCategories.length > 0">
            <div class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-2">
              Categorie predefinite
            </div>
            <div
              v-for="cat in defaultCategories"
              :key="cat.id"
              class="border border-gray-200 dark:border-gray-700 rounded-xl overflow-hidden"
            >
              <div
                class="flex items-center gap-2 p-3 bg-gray-50 dark:bg-gray-800 cursor-pointer"
                @click="toggleCategory(cat.id)"
              >
                <span class="text-lg w-6 shrink-0 text-center">{{ cat.icon }}</span>
                <div class="flex-1 min-w-0">
                  <span class="font-medium text-gray-900 dark:text-white truncate block">{{ cat.name }}</span>
                  <span class="text-xs text-gray-500 dark:text-gray-400">{{ cat.subcategories?.length || 0 }} sottocategorie</span>
                </div>
                <div class="flex items-center shrink-0">
                  <button
                    v-if="isAdmin"
                    @click.stop="startAddSubcategory(cat)"
                    class="p-1.5 text-blue-500 hover:text-blue-700 dark:hover:text-blue-400"
                    aria-label="Aggiungi sottocategoria"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                    </svg>
                  </button>
                  <button
                    v-if="isAdmin"
                    @click.stop="deleteCategory(cat)"
                    class="p-1.5 text-red-400 hover:text-red-600 dark:hover:text-red-400"
                    aria-label="Elimina categoria"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                  <svg
                    :class="['w-4 h-4 ml-1 text-gray-400 transition-transform', expandedCategories.has(cat.id) ? 'rotate-180' : '']"
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
                      class="flex-1 px-2 py-2 text-base border border-gray-200 dark:border-gray-700 rounded
                             bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                             focus:outline-none focus:ring-1 focus:ring-blue-500"
                      @keyup.enter="saveSubcategory(cat.id)"
                      ref="subcategoryInput"
                    />
                    <button @click="saveSubcategory(cat.id)" class="px-3 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-700">Aggiungi</button>
                    <button @click="addSubcategoryForCat = null; newSubcategoryName = ''" class="px-3 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800">Annulla</button>
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
                    class="p-2 text-red-400 hover:text-red-600 opacity-60 hover:opacity-100"
                    aria-label="Elimina sottocategoria"
                  >
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Personal categories -->
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
                class="flex items-center gap-2 p-3 bg-gray-50 dark:bg-gray-800 cursor-pointer"
                @click="toggleCategory(cat.id)"
              >
                <span class="text-lg w-6 shrink-0 text-center">{{ cat.icon }}</span>
                <div class="flex-1 min-w-0">
                  <span class="font-medium text-gray-900 dark:text-white truncate block">{{ cat.name }}</span>
                  <span class="text-xs text-gray-500 dark:text-gray-400">{{ cat.subcategories?.length || 0 }} sottocategorie</span>
                </div>
                <div class="flex items-center shrink-0">
                  <button
                    @click.stop="startAddSubcategory(cat)"
                    class="p-1.5 text-blue-500 hover:text-blue-700 dark:hover:text-blue-400"
                    aria-label="Aggiungi sottocategoria"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                    </svg>
                  </button>
                  <button
                    @click.stop="deleteCategory(cat)"
                    class="p-1.5 text-red-400 hover:text-red-600 dark:hover:text-red-400"
                    aria-label="Elimina categoria"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                  <svg
                    :class="['w-4 h-4 ml-1 text-gray-400 transition-transform', expandedCategories.has(cat.id) ? 'rotate-180' : '']"
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
                      class="flex-1 px-2 py-2 text-base border border-gray-200 dark:border-gray-700 rounded
                             bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                             focus:outline-none focus:ring-1 focus:ring-blue-500"
                      @keyup.enter="saveSubcategory(cat.id)"
                    />
                    <button @click="saveSubcategory(cat.id)" class="px-3 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-700">Aggiungi</button>
                    <button @click="addSubcategoryForCat = null; newSubcategoryName = ''" class="px-3 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800">Annulla</button>
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
                    class="p-2 text-red-400 hover:text-red-600 opacity-60 hover:opacity-100"
                    aria-label="Elimina sottocategoria"
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
    </div>

    <!-- Tab: Dati -->
    <div v-show="activeTab === 'dati'" class="space-y-4">
      <Card class="p-6">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Backup &amp; Dati</h2>

        <div class="space-y-6">
          <!-- Export -->
          <div>
            <h3 class="font-medium text-gray-900 dark:text-white mb-1">Esporta Dati</h3>
            <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
              Scarica i tuoi dati in formato JSON per backup o migrazione.
            </p>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <Button @click="doExport('all')" :disabled="exportLoading" class="w-full">
                {{ exportLoading === 'all' ? 'Esportazione...' : 'Esporta Tutto' }}
              </Button>
              <Button @click="doExport('expenses')" :disabled="exportLoading" variant="secondary" class="w-full">
                {{ exportLoading === 'expenses' ? 'Esportazione...' : 'Esporta Spese' }}
              </Button>
              <Button @click="doExport('utilities')" :disabled="exportLoading" variant="secondary" class="w-full">
                {{ exportLoading === 'utilities' ? 'Esportazione...' : 'Esporta Utenze' }}
              </Button>
              <Button @click="doExport('projects')" :disabled="exportLoading" variant="secondary" class="w-full">
                {{ exportLoading === 'projects' ? 'Esportazione...' : 'Esporta Progetti' }}
              </Button>
            </div>
          </div>

          <!-- Import -->
          <div class="border-t border-gray-200 dark:border-gray-700 pt-6">
            <h3 class="font-medium text-gray-900 dark:text-white mb-1">Importa Dati</h3>
            <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
              Carica un file di backup JSON per ripristinare o aggiungere dati.
            </p>

            <div
              @dragover.prevent
              @dragenter.prevent="isDragging = true"
              @dragleave.prevent="isDragging = false"
              @drop.prevent="handleFileDrop"
              :class="[
                'border-2 border-dashed rounded-xl p-8 text-center transition-colors cursor-pointer',
                isDragging
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                  : 'border-gray-300 dark:border-gray-600 hover:border-blue-400 dark:hover:border-blue-500'
              ]"
              @click="$refs.fileInput.click()"
            >
              <input ref="fileInput" type="file" accept=".json" class="hidden" @change="handleFileSelect" />
              <div class="text-3xl mb-2">📁</div>
              <p class="text-sm text-gray-600 dark:text-gray-400">
                Trascina il file JSON oppure <span class="text-blue-600 dark:text-blue-400 underline">seleziona</span>
              </p>
              <p v-if="selectedFile" class="text-sm font-medium text-gray-800 dark:text-gray-200 mt-2">
                {{ selectedFile.name }}
              </p>
            </div>

            <div v-if="selectedFile" class="mt-4 space-y-3">
              <div class="flex items-start gap-2 p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg text-sm text-yellow-800 dark:text-yellow-200">
                <span class="shrink-0 font-bold">!</span>
                <span>L'importazione <strong>aggiunge</strong> nuovi dati senza sovrascrivere quelli esistenti.</span>
              </div>
              <Button @click="doImport" :disabled="importLoading" class="w-full">
                {{ importLoading ? 'Importazione in corso...' : 'Importa Dati' }}
              </Button>
            </div>
          </div>
        </div>
      </Card>
    </div>

    <!-- Avatar Crop Modal -->
    <AvatarCropModal
      v-if="showCropModal"
      :image-src="cropImageSrc"
      @close="showCropModal = false; cropImageSrc = null"
      @cropped="onAvatarCropped"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { templatesAPI, categoriesAPI, exportAPI, authAPI, adminAPI, avatarAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import AvatarCropModal from '@/components/common/AvatarCropModal.vue'
import Button from '@/components/common/Button.vue'
import Input from '@/components/common/Input.vue'
import apiClient from '@/api/client'

const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

// Tabs
const activeTab = ref('famiglia')
const tabs = [
  { id: 'famiglia',   label: 'Famiglia',    icon: '👥' },
  { id: 'proprieta',  label: 'Proprietà',   icon: '🏠' },
  { id: 'preferenze', label: 'Preferenze',  icon: '⚙️' },
  { id: 'categorie',  label: 'Categorie',   icon: '🏷️' },
  { id: 'dati',       label: 'Dati',        icon: '📦' },
]

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
  const ok = await confirm({
    title: 'Elimina categoria',
    message: `Eliminare la ${label} "${cat.name}"? Anche tutte le sue sottocategorie verranno eliminate.`,
    confirmText: 'Elimina',
    variant: 'danger'
  })
  if (!ok) return
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
  const ok = await confirm({
    title: 'Elimina sottocategoria',
    message: `Eliminare la sottocategoria "${sub.name}"?`,
    confirmText: 'Elimina',
    variant: 'danger'
  })
  if (!ok) return
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
    const currentUserId = authStore.user?.id
    householdMembers.value = data.filter(m => m.user_id !== currentUserId)

    const myMember = data.find(m => m.user_id === currentUserId)
    if (myMember) {
      currentUserMemberId.value = myMember.id
    }
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

    if (data.default_split_with_member_ids) {
      try {
        defaultSplitMemberIds.value = JSON.parse(data.default_split_with_member_ids)
      } catch (e) {
        defaultSplitMemberIds.value = []
      }
    }

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
  if (!currentPropertyId.value) return

  try {
    await apiClient.put(`/properties/${currentPropertyId.value}/settings`, {
      split_mode: splitMode.value
    })
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
    settingsStore.theme = preferences.value.theme
    settingsStore.currency = preferences.value.currency
    settingsStore.language = preferences.value.language
    settingsStore.dateFormat = preferences.value.date_format
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
  } catch (err) {
    console.error('Error adding member:', err)
  }
}

async function deleteMember(memberId) {
  const ok = await confirm({
    title: 'Elimina membro',
    message: 'Sei sicuro di voler eliminare questo membro?',
    confirmText: 'Elimina',
    variant: 'danger'
  })
  if (!ok) return

  try {
    await apiClient.delete(`/members/${memberId}`)
    defaultSplitMemberIds.value = defaultSplitMemberIds.value.filter(id => id !== memberId)
    await updateUserSettings()
    await fetchHouseholdMembers()
  } catch (err) {
    console.error('Error deleting member:', err)
    if (err.response?.data?.error) {
      window.$toast?.error(err.response.data.error)
    }
  }
}

async function deleteUserAccount(member) {
  const ok = await confirm({
    title: 'Elimina account utente',
    message: `Eliminare definitivamente l'account di "${member.name}"? Tutti i dati associati (spese, utenze, bollette, letture, progetti, template) verranno cancellati in modo irreversibile.`,
    confirmText: 'Elimina definitivamente',
    variant: 'danger'
  })
  if (!ok) return

  try {
    await adminAPI.deleteUser(member.user_id)
    window.$toast?.success(`Account di "${member.name}" eliminato con successo`)
    await fetchHouseholdMembers()
  } catch (err) {
    console.error('Error deleting user account:', err)
    window.$toast?.error(err.response?.data?.error || "Errore durante l'eliminazione dell'account")
  }
}

async function toggleAdminRole(member) {
  const newRole = member.user_role === 'admin' ? 'user' : 'admin'
  const action = newRole === 'admin' ? 'promuovere ad admin' : 'rimuovere il ruolo admin da'
  const ok = await confirm({
    title: newRole === 'admin' ? 'Promuovi ad admin' : 'Rimuovi ruolo admin',
    message: `Sei sicuro di voler ${action} "${member.name}"?`,
    confirmText: newRole === 'admin' ? 'Promuovi' : 'Rimuovi',
    variant: newRole === 'admin' ? 'primary' : 'danger'
  })
  if (!ok) return

  try {
    await adminAPI.setUserRole(member.user_id, newRole)
    window.$toast?.success(`Ruolo di "${member.name}" aggiornato a ${newRole}`)
    await fetchHouseholdMembers()
  } catch (err) {
    console.error('Error toggling admin role:', err)
    window.$toast?.error(err.response?.data?.error || 'Errore durante il cambio ruolo')
  }
}

// ── Avatar ──────────────────────────────────────────────────────────────────

const avatarUploading = ref(false)
const avatarInput = ref(null)
const showCropModal = ref(false)
const cropImageSrc = ref(null)

function onAvatarSelected(e) {
  const file = e.target.files[0]
  if (!file) return
  if (file.size > 5 * 1024 * 1024) {
    window.$toast?.error('Immagine troppo grande (max 5MB)')
    if (avatarInput.value) avatarInput.value.value = ''
    return
  }
  // Read file and open crop modal
  const reader = new FileReader()
  reader.onload = (ev) => {
    cropImageSrc.value = ev.target.result
    showCropModal.value = true
  }
  reader.readAsDataURL(file)
  if (avatarInput.value) avatarInput.value.value = ''
}

async function onAvatarCropped(file) {
  showCropModal.value = false
  cropImageSrc.value = null
  avatarUploading.value = true
  try {
    const { data } = await avatarAPI.upload(file)
    authStore.updateUser(data.user)
    window.$toast?.success('Foto profilo aggiornata')
  } catch (err) {
    window.$toast?.error(err.response?.data?.error || 'Errore durante il caricamento')
  } finally {
    avatarUploading.value = false
  }
}

async function removeAvatar() {
  const ok = await confirm({
    title: 'Rimuovi foto profilo',
    message: 'Sei sicuro di voler rimuovere la foto profilo?',
    confirmText: 'Rimuovi',
    variant: 'danger'
  })
  if (!ok) return
  try {
    const { data } = await avatarAPI.delete()
    authStore.updateUser(data.user)
    window.$toast?.success('Foto profilo rimossa')
  } catch (err) {
    window.$toast?.error('Errore durante la rimozione')
  }
}

function handleLogout() {
  authStore.logout()
  router.push('/login')
}

// ── Cambio password ─────────────────────────────────────────────────────────

const showChangePassword = ref(false)
const pwLoading = ref(false)
const pwError = ref(null)
const pwSuccess = ref(null)
const pwForm = ref({ current: '', newPw: '', confirm: '' })

async function handleChangePassword() {
  pwError.value = null
  pwSuccess.value = null

  if (!pwForm.value.current) {
    pwError.value = 'Inserisci la password attuale.'
    return
  }
  if (pwForm.value.newPw.length < 6) {
    pwError.value = 'La nuova password deve avere almeno 6 caratteri.'
    return
  }
  if (pwForm.value.newPw !== pwForm.value.confirm) {
    pwError.value = 'Le due password non coincidono.'
    return
  }

  pwLoading.value = true
  try {
    await authAPI.changePassword(pwForm.value.current, pwForm.value.newPw)
    pwSuccess.value = 'Password aggiornata con successo!'
    pwForm.value = { current: '', newPw: '', confirm: '' }
    setTimeout(() => { showChangePassword.value = false; pwSuccess.value = null }, 2000)
  } catch (err) {
    pwError.value = err.response?.data?.error || 'Errore durante il cambio password.'
  } finally {
    pwLoading.value = false
  }
}

// ── Backup & Dati ──────────────────────────────────────────────────────────

const exportLoading = ref(null)
const exportSuccess = ref(null)
const exportError = ref(null)
const importLoading = ref(false)
const importSuccess = ref(null)
const importError = ref(null)
const isDragging = ref(false)
const selectedFile = ref(null)
const fileInput = ref(null)

async function doExport(type) {
  exportLoading.value = type
  exportSuccess.value = null
  exportError.value = null
  try {
    const apiMap = {
      all: exportAPI.exportAll,
      expenses: exportAPI.exportExpenses,
      utilities: exportAPI.exportUtilities,
      projects: exportAPI.exportProjects,
    }
    const nameMap = {
      all: 'backup_completo',
      expenses: 'spese',
      utilities: 'utenze',
      projects: 'progetti',
    }
    const res = await apiMap[type]()
    const timestamp = new Date().toISOString().slice(0, 10)
    triggerDownload(res.data, `homelog_${nameMap[type]}_${timestamp}.json`)
    window.$toast?.success('File scaricato con successo!')
  } catch (err) {
    window.$toast?.error('Errore esportazione: ' + (err.response?.data?.error || err.message))
  } finally {
    exportLoading.value = null
  }
}

function triggerDownload(blob, filename) {
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  window.URL.revokeObjectURL(url)
}

function handleFileDrop(e) {
  isDragging.value = false
  const file = e.dataTransfer.files[0]
  if (file && file.name.endsWith('.json')) {
    selectedFile.value = file
    importError.value = null
  } else {
    importError.value = 'Seleziona un file .json valido.'
  }
}

function handleFileSelect(e) {
  const file = e.target.files[0]
  if (file) {
    selectedFile.value = file
    importError.value = null
  }
}

async function doImport() {
  if (!selectedFile.value) return
  importLoading.value = true
  importSuccess.value = null
  importError.value = null
  try {
    const text = await selectedFile.value.text()
    const data = JSON.parse(text)
    const res = await exportAPI.importData(data)
    const counts = res.data.imported || {}
    const summary = Object.entries(counts)
      .map(([k, v]) => `${v} ${k}`)
      .join(', ')
    window.$toast?.success(`Importazione completata: ${summary || 'nessun dato'}.`)
    selectedFile.value = null
    if (fileInput.value) fileInput.value.value = ''
    setTimeout(() => { window.location.reload() }, 2000)
  } catch (err) {
    if (err instanceof SyntaxError) {
      window.$toast?.error('Il file non è un JSON valido.')
    } else {
      window.$toast?.error(err.response?.data?.error || 'Errore importazione: ' + err.message)
    }
  } finally {
    importLoading.value = false
  }
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
