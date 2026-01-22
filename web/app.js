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

async function init() {
    showLoader();
    
    try {
        const response = await fetch('/api/artists');
        if (!response.ok) {
            throw new Error('Erreur lors du chargement des artistes');
        }
        
        artists = await response.json();
        
        extractAllLocations();
        
        initializeFilters();
        
        showArtistList();
        applyFilters();
    } catch (error) {
        console.error('Erreur:', error);
        showError(error.message);
    }
}

function extractAllLocations() {
    const locationSet = new Set();
    
    artists.forEach(artist => {
        if (artist.locations && Array.isArray(artist.locations)) {
            artist.locations.forEach(loc => {
                const normalized = normalizeLocation(loc);
                locationSet.add(normalized);
            });
        }
    });
    
    allLocations = Array.from(locationSet).sort();
    filters.locations = [...allLocations];
}

function normalizeLocation(loc) {
    return loc.split('-')
        .map(part => part.charAt(0).toUpperCase() + part.slice(1))
        .join(', ');
}

function initializeFilters() {
    const creationYears = artists.map(a => a.creationDate);
    const albumYears = artists.map(a => a.firstAlbumYear).filter(y => y > 0);
    
    const minCreation = Math.min(...creationYears);
    const maxCreation = Math.max(...creationYears);
    const minAlbum = Math.min(...albumYears);
    const maxAlbum = Math.max(...albumYears);
    
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
    
    renderLocationCheckboxes(allLocations);
}

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

function applyFilters() {
    const query = filters.search.toLowerCase().trim();
    
    console.log('Debut du filtrage avec:', filters);
    
    const filtered = artists.filter((artist) => {
        const matchesSearch = !query || 
            artist.name.toLowerCase().includes(query) ||
            artist.members.some((member) => member.toLowerCase().includes(query));
        
        const matchesCreationDate = artist.creationDate >= filters.creationDateMin && 
                                   artist.creationDate <= filters.creationDateMax;
        
        const matchesFirstAlbum = artist.firstAlbumYear >= filters.firstAlbumMin && 
                                 artist.firstAlbumYear <= filters.firstAlbumMax;
        
        const memberCount = artist.members.length;
        const matchesMembers = filters.members.includes(memberCount) || 
                              (memberCount >= 6 && filters.members.includes('6+'));
        
        const matchesLocation = !artist.locations || 
                               artist.locations.length === 0 ||
                               artist.locations.some(loc => {
                                   const normalized = normalizeLocation(loc);
                                   return filters.locations.includes(normalized);
                               });
        
        const matches = matchesSearch && matchesCreationDate && matchesFirstAlbum && 
               matchesMembers && matchesLocation;
        
        if (!matches && artist.name.toLowerCase().includes('queen')) {
            console.log('Queen filtre:', {
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
    
    console.log(`${filtered.length} artistes apres filtrage`);

    renderArtistGrid(filtered);
}

function showLoader() {
    loader.classList.remove('hidden');
    artistListPage.classList.add('hidden');
    artistDetailPage.classList.add('hidden');
    errorMessage.classList.add('hidden');
}

function showArtistList() {
    loader.classList.add('hidden');
    artistListPage.classList.remove('hidden');
    artistDetailPage.classList.add('hidden');
    errorMessage.classList.add('hidden');
    currentArtist = null;
}

function showArtistDetail(artist) {
    currentArtist = artist;
    loader.classList.add('hidden');
    artistListPage.classList.add('hidden');
    artistDetailPage.classList.remove('hidden');
    errorMessage.classList.add('hidden');
    renderArtistDetail(artist);
}

function showError(message) {
    loader.classList.add('hidden');
    artistListPage.classList.add('hidden');
    artistDetailPage.classList.add('hidden');
    errorMessage.classList.remove('hidden');
    errorText.textContent = message;
}

function renderArtistGrid(artistList) {
    artistGrid.innerHTML = '';
    
    if (!artistList.length) {
        artistGrid.innerHTML = '<div class="empty-state">Aucun artiste ne correspond a votre recherche.</div>';
        updateStats(artistList);
        return;
    }

    artistList.forEach((artist) => {
        const card = document.createElement('div');
        card.className = 'artist-card';
        card.onclick = () => showArtistDetail(artist);

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
                <p class="muted-row">${displayLocations.length > 0 ? displayLocations.join(' - ') : 'Aucune localisation'}</p>
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

function renderArtistDetail(artist) {
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
                <span class="pill small">Cree en ${artist.creationDate}</span>
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

searchInput.addEventListener('input', () => {
    filters.search = searchInput.value;
    applyFilters();
});

filterToggle.addEventListener('click', () => {
    filterPanel.classList.toggle('hidden');
    filterToggle.classList.toggle('active');
});

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

locationSearch.addEventListener('input', (e) => {
    const query = e.target.value.toLowerCase();
    const filtered = allLocations.filter(loc => 
        loc.toLowerCase().includes(query)
    );
    renderLocationCheckboxes(filtered);
});

applyFiltersBtn.addEventListener('click', () => {
    filters.creationDateMin = parseInt(creationMinInput.value);
    filters.creationDateMax = parseInt(creationMaxInput.value);
    
    filters.firstAlbumMin = parseInt(albumMinInput.value);
    filters.firstAlbumMax = parseInt(albumMaxInput.value);
    
    filters.members = [];
    memberFilters.forEach(cb => {
        if (cb.checked) {
            const val = cb.value;
            filters.members.push(val === '6+' ? '6+' : parseInt(val));
        }
    });
    
    filters.locations = [];
    
    const searchQuery = locationSearch.value.toLowerCase().trim();
    
    if (searchQuery === '') {
        document.querySelectorAll('.location-filter').forEach(cb => {
            if (cb.checked) {
                filters.locations.push(cb.value);
            }
        });
    } else {
        allLocations.forEach(loc => {
            const checkbox = Array.from(document.querySelectorAll('.location-filter'))
                .find(cb => cb.value === loc);
            
            if (checkbox) {
                if (checkbox.checked) {
                    filters.locations.push(loc);
                }
            } else {
                if (filters.locations.includes(loc)) {
                    filters.locations.push(loc);
                }
            }
        });
    }
    
    console.log('Filtres appliques:', filters);
    
    applyFilters();
});

resetFiltersBtn.addEventListener('click', () => {
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

backButton.addEventListener('click', showArtistList);

document.addEventListener('keydown', (e) => {
    if (e.ctrlKey && e.key === 'f') {
        e.preventDefault();
        searchInput.focus();
    }
    
    if (e.key === 'Escape') {
        if (currentArtist) {
            showArtistList();
        }
    }
    
    if (e.ctrlKey && e.key === 'q') {
        e.preventDefault();
        if (confirm('Voulez-vous vraiment quitter ?')) {
            window.close();
        }
    }
});

function updateStats(list) {
    if (!list.length) {
        resultCount.textContent = '0';
        statAvgYear.textContent = '-';
        statMembers.textContent = '-';
        statOldest.textContent = '-';
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

window.addEventListener('DOMContentLoaded', init);
