/**
 * vmprober VMUI-style JavaScript utilities
 */

// Debounce utility
function debounce(func, delay) {
    let timer;
    return function (...args) {
        clearTimeout(timer);
        timer = setTimeout(() => func.apply(this, args), delay);
    };
}

// URL parameter management
function setParamURL(key, value) {
    const url = new URL(location.href);
    if (value) {
        url.searchParams.set(key, value);
    } else {
        url.searchParams.delete(key);
    }
    window.history.replaceState(null, null, url.toString());
}

function getParamURL(key) {
    return new URL(location.href).searchParams.get(key);
}

// Table filtering
function filterTable(searchPhrase, tableSelector = '.vm-table') {
    document.querySelectorAll(`${tableSelector} tbody tr`).forEach((row) => {
        const text = row.innerText.toLowerCase();
        row.style.display = !searchPhrase || text.includes(searchPhrase) ? '' : 'none';
    });
}

// Search handler
function handleSearch() {
    const searchBox = document.getElementById('search');
    if (!searchBox) return;
    
    const searchPhrase = searchBox.value.toLowerCase();
    filterTable(searchPhrase);
    setParamURL('search', searchPhrase);
}

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
    const searchBox = document.getElementById('search');
    
    if (searchBox) {
        searchBox.addEventListener('input', debounce(handleSearch, 200));
        
        // Restore search from URL
        const savedSearch = getParamURL('search');
        if (savedSearch) {
            searchBox.value = savedSearch;
            handleSearch();
        }
    }
});

