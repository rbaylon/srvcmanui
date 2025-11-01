function confirmDelete() {
    if (confirm("Are you sure you want to delete this item?")) {
        console.log("Item deleted!"); 
    } else {
        console.log("Deletion cancelled.");
    }
}