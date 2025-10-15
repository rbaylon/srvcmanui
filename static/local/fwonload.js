// load routers
document.addEventListener('DOMContentLoaded', function() {
    const myField = document.getElementById('basic-form-pfconfig');
    getPfconfig(encodeURIComponent(myField.value));
});