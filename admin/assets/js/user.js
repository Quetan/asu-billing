function getUser(userID) {
    fetch("/users/" + userID).then((res) => { return res.json() }).then((user) => {
        document.querySelector("#name").append(user.name);
        document.querySelector("#agreement").append(user.agreement);
        document.querySelector("#login").append(user.login);
        document.querySelector("#tariff").append(user.tariff.name);
        document.querySelector("#innerIP").append(user.inner_ip);
        document.querySelector("#extIP").append(user.ext_ip);
        document.querySelector("#phone").append(user.phone);
        document.querySelector("#room").append(user.room);
        document.querySelector("#connectionPlace").append(user.connection_place);
        const d = new Date(user.expired_date);
        if (user.activity === true) {
            const expiredDate = d.getDay() + "." + d.getMonth() + "." + d.getFullYear();
            document.querySelector("#expiredDate").append(expiredDate);
        } else {
            document.querySelector("#expiredDate").parentElement.remove();
        }
        document.querySelector("#balance").append(user.balance);
    })
}

window.onload = () => {
    const urlParams = new URLSearchParams(window.location.search);
    const userID = urlParams.get("id");
    getUser(userID);
}