function genericDurationHumanize(duration) {
    if (duration.asMilliseconds() > 0) return "in " + duration.humanize();
    return duration.humanize() + " ago";
}

var TIME_ZERO = "0001-01-01T00:00:00Z";
function isTimeZero(t) {
    return t === TIME_ZERO
}

$(function(){
    var API_URL = "http://127.0.0.1:8080/api/v1/";

    //
    // App-level Stats
    //
    $appDiv = $(".statsSection");
    function createStatBox(data, label) {
        var $box = $("<div class='pure-u-3-24 statsBox'></div>");

        if (!isNaN(Date.parse(data))) {
            if (isTimeZero(data)) return;

            data = genericDurationHumanize(
                moment.duration(moment(data).diff(Date.now()))
            );
        }

        $box.html([
            "<h3>", data, "</h3>",
            "<h4>", label, "</h4>",
        ].join(""));

        $appDiv.append($box);
    }

    function getStats(callback) {
        var statsUrl = API_URL + "stats/",
            createdAt = "CreatedAt";

        $.get(statsUrl, function(d) {
            $('.statsBox').remove();
            var statsKeys = Object.keys(d.Stats);
            for (var i = 0; i < statsKeys.length; i++) {
                if (statsKeys[i] === createdAt) continue;

                createStatBox(d.Stats[statsKeys[i]], statsKeys[i]);
            }
        });

    }

    getStats();
    setInterval(getStats, 1000 * 5); // Every 5 seconds


});
