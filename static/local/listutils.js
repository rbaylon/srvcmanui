function reloadFeatherIcons() {
    if (typeof feather !== 'undefined') {
        feather.replace();
    }
}

const speedTable = new DataTable('#records-speed', {
    pageLength: -1
});
speedTable.on('draw', reloadFeatherIcons);

const recordsTable = new DataTable('#records', {
    pageLength: 100
});
recordsTable.on('draw', reloadFeatherIcons);
async function getData(path) {
    try {
        const response = await fetch(path);
        if (!response.ok) {
            throw new Error(`Response status: ${response.status}`);
        }
    } catch (error) {
        console.error(error.message);
    }
}

function confirmDelete(path) {
    if (confirm("Are you sure you want to delete this item?")) {
        getData(path);
        console.log("Item deleted!");
    } else {
        console.log("Deletion cancelled.");
    }
    location.reload();
}