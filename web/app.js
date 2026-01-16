// État de l'application
let artists = [];
let allLocations = [];
let currentArtist = null;
let filters = {
    search: '',
    creationDateMin: 1950,
    creationDateMax: 2024,
    firstAlbumMin: 1950,
    firstAlbumMax: 2024,
    members: [1, 2, 3, 4, 5, '6+'],
    locations: []
};

// Éléments du DOM
const loader = document.getElementById('loader');
const artistListPage = document.getElementById('artist-list-page');
const artistDetailPage = document.getElementById('artist-detail-page');
const errorMessage = document.getElementById('error-message');
const errorText = document.getElementById('error-text');
const artistGrid = document.getElementById('artist-grid');
const artistDetail = document.getElementById('artist-detail');
const searchInput = document.getElementById('search-input');
const backButton = document.getElementById('back-button');
const resultCount = document.getElementById('result-count');
const statAvgYear = document.getElementById('stat-avg-year');
const statMembers = document.getElementById('stat-members');
const statOldest = document.getElementById('stat-oldest');

// Éléments des filtres
const filterToggle = document.getElementById('filter-toggle');
const filterPanel = document.getElementById('filter-panel');
const creationMinInput = document.getElementById('creation-min');
const creationMaxInput = document.getElementById('creation-max');
const creationSliderMin = document.getElementById('creation-slider-min');
const creationSliderMax = document.getElementById('creation-slider-max');
const albumMinInput = document.getElementById('album-min');
const albumMaxInput = document.getElementById('album-max');
const albumSliderMin = document.getElementById('album-slider-min');
const albumSliderMax = document.getElementById('album-slider-max');
const memberFilters = document.querySelectorAll('.member-filter');
const locationSearch = document.getElementById('location-search');
const locationCheckboxes = document.getElementById('location-checkboxes');
const resetFiltersBtn = document.getElementById('reset-filters');
const applyFiltersBtn = document.getElementById('apply-filters');

// Initialisation de l'application
async function init() {
    showLoader();
    
    try {
        // Appel API réel
        const response = await fetch('/api/artists');
        if (!response.ok) {
            throw new Error('Erreur lors du chargement des artistes');
        }
        
        artists = await response.json();
        
        // Extraire tous les lieux uniques et normaliser
        extractAllLocations();
        
        // Initialiser l'UI des filtres
        initializeFilters();
        
        showArtistList();
        applyFilters();
    } catch (error) {
        console.error('Erreur:', error);
        showError(error.message);
    }
}

// Extraire tous les lieux uniques de tous les artistes
function extractAllLocations() {
    const locationSet = new Set();
    
    artists.forEach(artist => {
        if (artist.locations && Array.isArray(artist.locations)) {
            artist.locations.forEach(loc => {
                // Normaliser le lieu (gérer les formats "ville-pays")
                const normalized = normalizeLocation(loc);
                locationSet.add(normalized);
            });
        }
    });
    
    allLocations = Array.from(locationSet).sort();
    filters.locations = [...allLocations]; // Par défaut, tous sélectionnés
}

// Normaliser un lieu (exemple: "seattle-washington-usa" -> "Seattle, Washington, USA")
function normalizeLocation(loc) {
    return loc.split('-')
        .map(part => part.charAt(0).toUpperCase() + part.slice(1))
        .join(', ');
}

// Initialiser les éléments UI des filtres
function initializeFilters() {
    // Trouver les années min/max réelles
    const creationYears = artists.map(a => a.creationDate);
    const albumYears = artists.map(a => a.firstAlbumYear).filter(y => y > 0);
    
    const minCreation = Math.min(...creationYears);
    const maxCreation = Math.max(...creationYears);
    const minAlbum = Math.min(...albumYears);
    const maxAlbum = Math.max(...albumYears);
    
    // Mettre à jour les limites des sliders
    [creationMinInput, creationSliderMin].forEach(el => {
        el.min = minCreation;
        el.max = maxCreation;
        el.value = minCreation;
    });
    
    [creationMaxInput, creationSliderMax].forEach(el => {
        el.min = minCreation;
        el.max = maxCreation;
        el.value = maxCreation;
    });
    
    [albumMinInput, albumSliderMin].forEach(el => {
        el.min = minAlbum;
        el.max = maxAlbum;
        el.value = minAlbum;
    });
    
    [albumMaxInput, albumSliderMax].forEach(el => {
        el.min = minAlbum;
        el.max = maxAlbum;
        el.value = maxAlbum;
    });
    
    filters.creationDateMin = minCreation;
    filters.creationDateMax = maxCreation;
    filters.firstAlbumMin = minAlbum;
    filters.firstAlbumMax = maxAlbum;
    
    // Générer les checkboxes pour les lieux
    renderLocationCheckboxes(allLocations);
}

// Générer les checkboxes pour les lieux
function renderLocationCheckboxes(locations) {
    locationCheckboxes.innerHTML = '';
    
    locations.forEach(loc => {
        const label = document.createElement('label');
        label.className = 'checkbox-label';
        
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.value = loc;
        checkbox.checked = filters.locations.includes(loc);
        checkbox.className = 'location-filter';
        
        const span = document.createElement('span');
        span.textContent = loc;
        
        label.appendChild(checkbox);
        label.appendChild(span);
        locationCheckboxes.appendChild(label);
    });
}

// Applique tous les filtres actifs
function applyFilters() {
    const query = filters.search.toLowerCase().trim();
    
    console.log('Début du filtrage avec:', filters);
    
    const filtered = artists.filter((artist) => {
        // Filtre de recherche textuelle
        const matchesSearch = !query || 
            artist.name.toLowerCase().includes(query) ||
            artist.members.some((member) => member.toLowerCase().includes(query));
        
        // Filtre par date de création
        const matchesCreationDate = artist.creationDate >= filters.creationDateMin && 
                                   artist.creationDate <= filters.creationDateMax;
        
        // Filtre par année du premier album
        const matchesFirstAlbum = artist.firstAlbumYear >= filters.firstAlbumMin && 
                                 artist.firstAlbumYear <= filters.firstAlbumMax;
        
        // Filtre par nombre de membres
        const memberCount = artist.members.length;
        const matchesMembers = filters.members.includes(memberCount) || 
                              (memberCount >= 6 && filters.members.includes('6+'));
        
        // Filtre par lieu de concert
        // Un artiste correspond si AU MOINS UN de ses lieux est dans la liste des lieux sélectionnés
        const matchesLocation = !artist.locations || 
                               artist.locations.length === 0 ||
                               artist.locations.some(loc => {
                                   const normalized = normalizeLocation(loc);
                                   return filters.locations.includes(normalized);
                               });
        
        const matches = matchesSearch && matchesCreationDate && matchesFirstAlbum && 
               matchesMembers && matchesLocation;
        
        if (!matches && artist.name.toLowerCase().includes('queen')) {
            console.log('Queen filtré:', {
                matchesSearch,
                matchesCreationDate,
                matchesFirstAlbum,
                matchesMembers,
                matchesLocation,
                locations: artist.locations,
                filterLocations: filters.locations
            });
        }
        
        return matches;
    });
    
    console.log(`${filtered.length} artistes après filtrage`);

    renderArtistGrid(filtered);
}

// Afficher le loader
function showLoader() {
    loader.classList.remove('hidden');
    artistListPage.classList.add('hidden');
    artistDetailPage.classList.add('hidden');
    errorMessage.classList.add('hidden');
}

// Afficher la liste des artistes
function showArtistList() {
    loader.classList.add('hidden');
    artistListPage.classList.remove('hidden');
    artistDetailPage.classList.add('hidden');
    errorMessage.classList.add('hidden');
    currentArtist = null;
}

// Afficher la page de détail
function showArtistDetail(artist) {
    currentArtist = artist;
    loader.classList.add('hidden');
    artistListPage.classList.add('hidden');
    artistDetailPage.classList.remove('hidden');
    errorMessage.classList.add('hidden');
    renderArtistDetail(artist);
}

// Afficher une erreur
function showError(message) {
    loader.classList.add('hidden');
    artistListPage.classList.add('hidden');
    artistDetailPage.classList.add('hidden');
    errorMessage.classList.remove('hidden');
    errorText.textContent = message;
}

// Générer la grille d'artistes
function renderArtistGrid(artistList) {
    artistGrid.innerHTML = '';
    
    if (!artistList.length) {
        artistGrid.innerHTML = '<div class="empty-state">Aucun artiste ne correspond à votre recherche.</div>';
        updateStats(artistList);
        return;
    }

    artistList.forEach((artist) => {
        const card = document.createElement('div');
        card.className = 'artist-card';
        card.onclick = () => showArtistDetail(artist);

        // Préparer les lieux normalisés
        let displayLocations = [];
        if (artist.locations && artist.locations.length > 0) {
            displayLocations = artist.locations.slice(0, 3).map(loc => normalizeLocation(loc));
        }

        card.innerHTML = `
            <div class="artist-media">
                <img src="${artist.image}" alt="${artist.name}" onerror="this.src='https://via.placeholder.com/400x300?text=${encodeURIComponent(artist.name)}'">
                <span class="badge">${artist.creationDate}</span>
            </div>
            <div class="artist-body">
                <div class="artist-title-row">
                    <h3>${artist.name}</h3>
                    <span class="pill small">${artist.members.length} ${artist.members.length > 1 ? 'membres' : 'membre'}</span>
                </div>
                <p class="muted-row">${displayLocations.length > 0 ? displayLocations.join(' • ') : 'Aucune localisation'}</p>
                <div class="tag-row">
                    ${artist.members.slice(0, 3).map((member) => `<span class="chip">${member}</span>`).join('')}
                    ${artist.members.length > 3 ? `<span class="chip">+${artist.members.length - 3}</span>` : ''}
                </div>
            </div>
        `;

        artistGrid.appendChild(card);
    });

    updateStats(artistList);
}

// Générer la page de détail
function renderArtistDetail(artist) {
    // Préparer les dates et lieux depuis datesLocations
    let locationsHTML = '';
    let datesHTML = '';
    
    if (artist.datesLocations && Object.keys(artist.datesLocations).length > 0) {
        const locations = Object.keys(artist.datesLocations);
        locationsHTML = locations.map(loc => 
            `<div class="tag-card">${normalizeLocation(loc)}</div>`
        ).join('');
        
        const allDates = [];
        Object.values(artist.datesLocations).forEach(dates => {
            allDates.push(...dates);
        });
        datesHTML = allDates.slice(0, 10).map(date => 
            `<div class="tag-card">${date}</div>`
        ).join('');
    } else if (artist.locations && artist.locations.length > 0) {
        locationsHTML = artist.locations.map(loc => 
            `<div class="tag-card">${loc}</div>`
        ).join('');
    }
    
    artistDetail.innerHTML = `
        <div class="detail-visual">
            <div class="detail-glow"></div>
            <img src="${artist.image}" alt="${artist.name}" onerror="this.src='https://via.placeholder.com/600x420?text=${encodeURIComponent(artist.name)}'">
        </div>
        
        <div class="detail-info">
            <p class="eyebrow">Profil artiste</p>
            <h2>${artist.name}</h2>
            
            <div class="pill-row">
                <span class="pill small">Créé en ${artist.creationDate}</span>
                <span class="pill small">1er album : ${artist.firstAlbum}</span>
                <span class="pill small">${artist.members.length} membres</span>
            </div>

            <div class="info-grid">
                <div class="info-card">
                    <h3>Membres</h3>
                    <div class="list-chips">
                        ${artist.members.map((member) => `<span class="chip">${member}</span>`).join('')}
                    </div>
                </div>
                ${locationsHTML ? `
                <div class="info-card">
                    <h3>Lieux de concerts</h3>
                    <div class="grid-tags">
                        ${locationsHTML}
                    </div>
                </div>` : ''}
                ${datesHTML ? `
                <div class="info-card">
                    <h3>Dates de concerts</h3>
                    <div class="grid-tags">
                        ${datesHTML}
                    </div>
                </div>` : ''}
            </div>
        </div>
    `;
}

// Recherche d'artistes
searchInput.addEventListener('input', () => {
    filters.search = searchInput.value;
    applyFilters();
});

// Toggle du panneau de filtres
filterToggle.addEventListener('click', () => {
    filterPanel.classList.toggle('hidden');
    filterToggle.classList.toggle('active');
});

// Synchroniser les sliders et inputs pour la date de création
creationSliderMin.addEventListener('input', (e) => {
    const val = parseInt(e.target.value);
    if (val > parseInt(creationSliderMax.value)) {
        creationSliderMax.value = val;
        creationMaxInput.value = val;
    }
    creationMinInput.value = val;
});

creationSliderMax.addEventListener('input', (e) => {
    const val = parseInt(e.target.value);
    if (val < parseInt(creationSliderMin.value)) {
        creationSliderMin.value = val;
        creationMinInput.value = val;
    }
    creationMaxInput.value = val;
});

creationMinInput.addEventListener('input', (e) => {
    const val = parseInt(e.target.value);
    if (val > parseInt(creationMaxInput.value)) {
        creationMaxInput.value = val;
        creationSliderMax.value = val;
    }
    creationSliderMin.value = val;
});

creationMaxInput.addEventListener('input', (e) => {
    const val = parseInt(e.target.value);
    if (val < parseInt(creationMinInput.value)) {
        creationMinInput.value = val;
        creationSliderMin.value = val;
    }
    creationSliderMax.value = val;
});

// Synchroniser les sliders et inputs pour l'année du premier album
albumSliderMin.addEventListener('input', (e) => {
    const val = parseInt(e.target.value);
    if (val > parseInt(albumSliderMax.value)) {
        albumSliderMax.value = val;
        albumMaxInput.value = val;
    }
    albumMinInput.value = val;
});

albumSliderMax.addEventListener('input', (e) => {
    const val = parseInt(e.target.value);
    if (val < parseInt(albumSliderMin.value)) {
        albumSliderMin.value = val;
        albumMinInput.value = val;
    }
    albumMaxInput.value = val;
});

albumMinInput.addEventListener('input', (e) => {
    const val = parseInt(e.target.value);
    if (val > parseInt(albumMaxInput.value)) {
        albumMaxInput.value = val;
        albumSliderMax.value = val;
    }
    albumSliderMin.value = val;
});

albumMaxInput.addEventListener('input', (e) => {
    const val = parseInt(e.target.value);
    if (val < parseInt(albumMinInput.value)) {
        albumMinInput.value = val;
        albumSliderMin.value = val;
    }
    albumSliderMax.value = val;
});

// Recherche dans les lieux
locationSearch.addEventListener('input', (e) => {
    const query = e.target.value.toLowerCase();
    const filtered = allLocations.filter(loc => 
        loc.toLowerCase().includes(query)
    );
    renderLocationCheckboxes(filtered);
});

// Bouton appliquer les filtres
applyFiltersBtn.addEventListener('click', () => {
    // Récupérer les valeurs des filtres de date de création
    filters.creationDateMin = parseInt(creationMinInput.value);
    filters.creationDateMax = parseInt(creationMaxInput.value);
    
    // Récupérer les valeurs des filtres d'album
    filters.firstAlbumMin = parseInt(albumMinInput.value);
    filters.firstAlbumMax = parseInt(albumMaxInput.value);
    
    // Récupérer les membres sélectionnés
    filters.members = [];
    memberFilters.forEach(cb => {
        if (cb.checked) {
            const val = cb.value;
            filters.members.push(val === '6+' ? '6+' : parseInt(val));
        }
    });
    
    // Récupérer TOUS les lieux sélectionnés
    // Important: parcourir TOUS les lieux possibles, pas seulement ceux affichés
    filters.locations = [];
    
    // Méthode 1: Si aucune recherche de lieu n'est active, on peut lire directement les checkboxes
    const searchQuery = locationSearch.value.toLowerCase().trim();
    
    if (searchQuery === '') {
        // Pas de recherche active, on lit directement les checkboxes visibles
        document.querySelectorAll('.location-filter').forEach(cb => {
            if (cb.checked) {
                filters.locations.push(cb.value);
            }
        });
    } else {
        // Il y a une recherche active, donc on doit vérifier tous les lieux
        allLocations.forEach(loc => {
            // Chercher si ce lieu a une checkbox actuellement affichée
            const checkbox = Array.from(document.querySelectorAll('.location-filter'))
                .find(cb => cb.value === loc);
            
            if (checkbox) {
                // Le lieu est visible, on prend son état coché
                if (checkbox.checked) {
                    filters.locations.push(loc);
                }
            } else {
                // Le lieu n'est pas visible (filtré par la recherche)
                // On garde l'ancien état s'il était sélectionné
                if (filters.locations.includes(loc)) {
                    filters.locations.push(loc);
                }
            }
        });
    }
    
    console.log('Filtres appliqués:', filters);
    
    // Appliquer les filtres
    applyFilters();
});

// Bouton réinitialiser les filtres
resetFiltersBtn.addEventListener('click', () => {
    // Réinitialiser tous les filtres
    const creationYears = artists.map(a => a.creationDate);
    const albumYears = artists.map(a => a.firstAlbumYear).filter(y => y > 0);
    
    const minCreation = Math.min(...creationYears);
    const maxCreation = Math.max(...creationYears);
    const minAlbum = Math.min(...albumYears);
    const maxAlbum = Math.max(...albumYears);
    
    creationMinInput.value = minCreation;
    creationMaxInput.value = maxCreation;
    creationSliderMin.value = minCreation;
    creationSliderMax.value = maxCreation;
    
    albumMinInput.value = minAlbum;
    albumMaxInput.value = maxAlbum;
    albumSliderMin.value = minAlbum;
    albumSliderMax.value = maxAlbum;
    
    memberFilters.forEach(cb => cb.checked = true);
    
    document.querySelectorAll('.location-filter').forEach(cb => cb.checked = true);
    
    filters.creationDateMin = minCreation;
    filters.creationDateMax = maxCreation;
    filters.firstAlbumMin = minAlbum;
    filters.firstAlbumMax = maxAlbum;
    filters.members = [1, 2, 3, 4, 5, '6+'];
    filters.locations = [...allLocations];
    
    applyFilters();
});

// Bouton retour
backButton.addEventListener('click', showArtistList);

// Raccourcis clavier
document.addEventListener('keydown', (e) => {
    // Ctrl + F : Focus recherche
    if (e.ctrlKey && e.key === 'f') {
        e.preventDefault();
        searchInput.focus();
    }
    
    // ESC : Retour
    if (e.key === 'Escape') {
        if (currentArtist) {
            showArtistList();
        }
    }
    
    // Ctrl + Q : Quitter (fermer l'onglet)
    if (e.ctrlKey && e.key === 'q') {
        e.preventDefault();
        if (confirm('Voulez-vous vraiment quitter ?')) {
            window.close();
        }
    }
});

// Met à jour les stats dans le header
function updateStats(list) {
    if (!list.length) {
        resultCount.textContent = '0';
        statAvgYear.textContent = '–';
        statMembers.textContent = '–';
        statOldest.textContent = '–';
        return;
    }

    const totalMembers = list.reduce((acc, a) => acc + a.members.length, 0);
    const avgMembers = (totalMembers / list.length).toFixed(1);
    const avgYear = Math.round(list.reduce((acc, a) => acc + a.creationDate, 0) / list.length);
    const oldest = Math.min(...list.map((a) => a.creationDate));

    resultCount.textContent = `${list.length}`;
    statAvgYear.textContent = `~${avgYear}`;
    statMembers.textContent = `${avgMembers}`;
    statOldest.textContent = `Depuis ${oldest}`;
}

// Lancer l'application au chargement
window.addEventListener('DOMContentLoaded', init);
