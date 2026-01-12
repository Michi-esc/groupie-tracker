// État de l'application
let artists = [];
let currentArtist = null;
let activeDecade = 'all';

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
const decadeFilters = document.querySelectorAll('[data-decade]');

// Données de test (à remplacer par un appel API)
const getDummyArtists = () => {
    return [
        {
            id: 1,
            name: "Queen",
            members: ["Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"],
            creationDate: 1970,
            firstAlbum: "14-12-1973",
            image: "https://images.unsplash.com/photo-1493225457124-a3eb161ffa5f?w=400",
            locations: ["London", "Paris", "New York", "Tokyo", "Sydney"],
            concertDates: ["01-01-2024", "15-02-2024", "30-03-2024", "10-05-2024"]
        },
        {
            id: 2,
            name: "The Beatles",
            members: ["John Lennon", "Paul McCartney", "George Harrison", "Ringo Starr"],
            creationDate: 1960,
            firstAlbum: "22-03-1963",
            image: "https://images.unsplash.com/photo-1511671782779-c97d3d27a1d4?w=400",
            locations: ["Liverpool", "Hamburg", "New York", "Los Angeles"],
            concertDates: ["10-05-2024", "20-06-2024", "15-08-2024"]
        },
        {
            id: 3,
            name: "Pink Floyd",
            members: ["Roger Waters", "David Gilmour", "Nick Mason", "Richard Wright"],
            creationDate: 1965,
            firstAlbum: "05-08-1967",
            image: "https://images.unsplash.com/photo-1470225620780-dba8ba36b745?w=400",
            locations: ["London", "Los Angeles", "Berlin", "Amsterdam"],
            concertDates: ["01-07-2024", "20-07-2024"]
        },
        {
            id: 4,
            name: "Led Zeppelin",
            members: ["Robert Plant", "Jimmy Page", "John Paul Jones", "John Bonham"],
            creationDate: 1968,
            firstAlbum: "12-01-1969",
            image: "https://images.unsplash.com/photo-1501281668745-f7f57925c3b4?w=400",
            locations: ["London", "New York", "Chicago", "San Francisco"],
            concertDates: ["05-09-2024", "25-09-2024", "10-10-2024"]
        },
        {
            id: 5,
            name: "The Rolling Stones",
            members: ["Mick Jagger", "Keith Richards", "Charlie Watts", "Ronnie Wood"],
            creationDate: 1962,
            firstAlbum: "16-04-1964",
            image: "https://images.unsplash.com/photo-1459749411175-04bf5292ceea?w=400",
            locations: ["London", "Paris", "Madrid", "Rome", "Berlin"],
            concertDates: ["01-11-2024", "15-11-2024", "30-11-2024"]
        },
        {
            id: 6,
            name: "Nirvana",
            members: ["Kurt Cobain", "Krist Novoselic", "Dave Grohl"],
            creationDate: 1987,
            firstAlbum: "15-06-1989",
            image: "https://images.unsplash.com/photo-1498038432885-c6f3f1b912ee?w=400",
            locations: ["Seattle", "Portland", "Los Angeles", "New York"],
            concertDates: ["12-12-2024", "20-12-2024"]
        }
    ];
};

// Initialisation de l'application
async function init() {
    showLoader();
    
    try {
        // Simuler un chargement API
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        // TODO: Remplacer par un vrai appel API
        // const response = await fetch('/api/artists');
        // artists = await response.json();
        
        artists = getDummyArtists();
        
        showArtistList();
        applyFilters();
    } catch (error) {
        showError(error.message);
    }
}

// Applique recherche + filtre époque
function applyFilters() {
    const query = searchInput.value.toLowerCase().trim();
    const filtered = artists.filter((artist) => {
        const matchesQuery =
            artist.name.toLowerCase().includes(query) ||
            artist.members.some((member) => member.toLowerCase().includes(query));
        const matchesEra = matchesDecade(artist.creationDate);
        return matchesQuery && matchesEra;
    });

    renderArtistGrid(filtered);
}

function matchesDecade(year) {
    switch (activeDecade) {
        case '60s':
            return year >= 1960 && year < 1980;
        case '80s':
            return year >= 1980 && year < 2000;
        case 'modern':
            return year >= 2000;
        default:
            return true;
    }
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

        card.innerHTML = `
            <div class="artist-media">
                <img src="${artist.image}" alt="${artist.name}" onerror="this.src='https://via.placeholder.com/400x300?text=${encodeURIComponent(artist.name)}'">
                <span class="badge">${artist.creationDate}</span>
            </div>
            <div class="artist-body">
                <div class="artist-title-row">
                    <h3>${artist.name}</h3>
                    <span class="pill small">${artist.members.length} membres</span>
                </div>
                <p class="muted-row">${artist.locations.slice(0, 3).join(' • ') || 'Aucune localisation'}</p>
                <div class="tag-row">
                    ${artist.members.slice(0, 3).map((member) => `<span class="chip">${member}</span>`).join('')}
                </div>
            </div>
        `;

        artistGrid.appendChild(card);
    });

    updateStats(artistList);
}

// Générer la page de détail
function renderArtistDetail(artist) {
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
                <div class="info-card">
                    <h3>Lieux</h3>
                    <div class="grid-tags">
                        ${artist.locations.map((loc) => `<div class="tag-card">${loc}</div>`).join('')}
                    </div>
                </div>
                <div class="info-card">
                    <h3>Dates</h3>
                    <div class="grid-tags">
                        ${artist.concertDates.map((date) => `<div class="tag-card">${date}</div>`).join('')}
                    </div>
                </div>
            </div>
        </div>
    `;
}

// Recherche d'artistes
searchInput.addEventListener('input', () => {
    applyFilters();
});

decadeFilters.forEach((btn) => {
    btn.addEventListener('click', () => {
        activeDecade = btn.dataset.decade;
        decadeFilters.forEach((b) => b.classList.remove('active'));
        btn.classList.add('active');
        applyFilters();
    });
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
