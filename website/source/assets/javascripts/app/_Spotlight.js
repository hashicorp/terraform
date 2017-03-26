(function($) {
    "use strict";

    var pages = {};
    var visible = false;
    var lastText = null;

    $.ajax({
        url: '/sitemap.xml',
        dataType: 'xml'
    }).done(function(data) {
        pages = loadSitemap(data);
        $(function() {
            initSpotlight();
        });
    });

    function loadSitemap(data) {
        var nsSitemaps = "http://www.sitemaps.org/schemas/sitemap/0.9";
        var nsDcTerms = "http://purl.org/dc/terms/"

        var nodes = data.documentElement.getElementsByTagNameNS(nsSitemaps, 'url');
        nodes = Array.prototype.map.call(nodes, function(node) {
            var title = node.getElementsByTagNameNS(nsDcTerms, 'title')[0];
            if (!title) return null;
            var url = node.getElementsByTagNameNS(nsSitemaps, 'loc')[0].textContent;
            url = url.replace(/^http.*:\/\/.*?\//, '/');

            return {
                title: title.textContent,
                url: url
            };
        });
        var result = nodes.filter(function(x) { return !!x; });
        result.sort(function(a, b) {
            a = a.title.toLowerCase();
            b = b.title.toLowerCase();
            return a < b ? -1 : a > b ? 1 : 0;
        });
        return result;
    }


    function initSpotlight() {

        // Ctrl-P brings up the spotlight

        $(document).keyup(function(e) {
            if (e.altKey || e.metaKey || e.shiftKey) return;
            if (e.key.toLowerCase() == 'p' && e.ctrlKey) {
                handleShowRequest(e);
            }
        });

        // Esc dismisses it

        $(document).on('keyup', '#spotlight', function(e) {
            if (e.altKey || e.ctrlKey || e.shiftKey || e.metaKey) return;
            switch (e.keyCode) {
                case 27: // esc
                    handleEsc(e);
            }
        });

        // So too does losing focus

        $(document).on('focusout', '#spotlight', function(e) {
            handleFocusOut(this);
        });

        // A key press in the input box refreshes the results

        $(document).on('keyup', '#spotlight input', function(e) {
            handleKeyInInputBox();
        });
    }


    function handleShowRequest(e) {
        showSpotlight();
        updateSpotlight($('#spotlight input').val());
        e.preventDefault();
    }

    function handleEsc(e) {
        if (hideSpotlight()) {
            e.preventDefault();
        }
    }

    function handleFocusOut(spotlightElement) {
        // Wait till focus has finished moving before testing whether
        // or not to dismiss the spotlight.
        window.setTimeout(function() {
            if (!$.contains(spotlightElement, document.activeElement)) {
                hideSpotlight();
            }
        }, 1);
    }

    function handleKeyInInputBox() {
        var text = $("#spotlight input").val();
        if (text === lastText) return;
        lastText = text;
        updateSpotlight(text);
    }

    function showSpotlight() {
        if (visible) return;
        visible = true;
        $('body').prepend(
            '<div id="spotlight"><div id="spotlight-contents">' +
            '<input type="text" placeholder="Quick search" />' +
            '<div id="spotlight-results">' +
            '</div></div></div>');
        $('#spotlight input').focus();
    }

    function hideSpotlight() {
        if (!visible) return false;
        visible = false;
        lastText = null;
        $("#spotlight").hide(0, function() {
            $('#spotlight').remove();
        });
        return true;
    }

    function updateSpotlight(text) {
        var matches = getMatches(text);
        $('#spotlight-results a').remove();
        if(matches.length) {
            $('#spotlight-results').append(matches).show(0);
        }
        else {
            $('#spotlight-results').hide(0);
        }
    }

    function getMatches(text) {
        if (text.length < 2) return [];
        var links = [];
        for (var i = 0; i < pages.length; i++) {
            var p = pages[i];
            var ix = p.title.toLowerCase().indexOf(text.toLowerCase());
            if (ix >= 0) {
                var $element = $('<a>');
                $element.append(p.title.substr(0, ix));
                var $selection = $('<strong>');
                $selection.append(p.title.substr(ix, text.length));
                $element.append($selection);
                $element.append(p.title.substr(ix + text.length));

                var url = p.url;
                var isDataSource = /\/d\//.test(url);
                var isResource = /\/r\//.test(url);
                if (isDataSource) {
                    $element.append($('<em>(Data Source)</em>'));
                }
                if (isResource) {
                    $element.append($('<em>(Resource)</em>'));
                }

                $element.attr('href', url);
                links.push($element);
            }
        }

        return links;
    }

})(jQuery);