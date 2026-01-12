// État de l'application
let artists = [];
let currentArtist = null;

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
        renderArtistGrid(artists);
    } catch (error) {
        showError(error.message);
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
    
    artistList.forEach(artist => {
        const card = document.createElement('div');
        card.className = 'artist-card';
        card.onclick = () => showArtistDetail(artist);
        
        card.innerHTML = `
            <img src="${artist.image}" alt="${artist.name}" onerror="this.src='https://via.placeholder.com/400x200?text=${artist.name}'">
            <h3>${artist.name}</h3>
            <p class="year">Créé en ${artist.creationDate}</p>
        `;
        
        artistGrid.appendChild(card);
    });
}

// Générer la page de détail
function renderArtistDetail(artist) {
    artistDetail.innerHTML = `
        <img src="${artist.image}" alt="${artist.name}" onerror="this.src='https://via.placeholder.com/400x300?text=${artist.name}'">
        
        <h2>${artist.name}</h2>
        
        <div class="artist-info">
            <h3>📅 Informations</h3>
            <p><strong>Créé en :</strong> ${artist.creationDate}</p>
            <p><strong>Premier album :</strong> ${artist.firstAlbum}</p>
        </div>
        
        <div class="artist-info">
            <h3>👥 Membres</h3>
            <ul>
                ${artist.members.map(member => `<li>${member}</li>`).join('')}
            </ul>
        </div>
        
        <div class="artist-info">
            <h3>📍 Lieux de concert</h3>
            <div class="locations-list">
                ${artist.locations.map(loc => `<div class="location-item">${loc}</div>`).join('')}
            </div>
        </div>
        
        <div class="artist-info">
            <h3>🎫 Dates de concert</h3>
            <div class="dates-list">
                ${artist.concertDates.map(date => `<div class="date-item">${date}</div>`).join('')}
            </div>
        </div>
    `;
}

// Recherche d'artistes
searchInput.addEventListener('input', (e) => {
    const query = e.target.value.toLowerCase();
    const filtered = artists.filter(artist => 
        artist.name.toLowerCase().includes(query) ||
        artist.members.some(member => member.toLowerCase().includes(query))
    );
    renderArtistGrid(filtered);
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

// Lancer l'application au chargement
window.addEventListener('DOMContentLoaded', init);
