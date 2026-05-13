new DataTable('#records-speed', {
    paging: false
});
new DataTable('#records', {
    pageLength: 100
});
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