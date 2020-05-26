/*! Reef v7.3.0 | (c) 2020 Chris Ferdinandi | MIT License | http://github.com/cferdinandi/reef */
'use strict';

//
// Variables
//

// RegExp patterns
var regPatterns = {
	parameterRegex: /([:*])(\w+)/g,
	wildcardRegex: /\*/g,
	replaceVarRegex: '([^\/]+)',
	replaceWildcard: '(?:.*)',
	followedBySlashRegex: '(?:\/$|$)',
	matchRegexFlags: ''
};

// Check for pushState() support
var support = !!window.history && !!window.history.pushState;


//
// Methods
//

/**
 * Get the URL parameters
 * source: https://css-tricks.com/snippets/javascript/get-url-variables/
 * @param  {String} url The URL
 * @return {Object}     The URL parameters
 */
var getParams = function (url) {
	var params = {};
	var query = url.search.substring(1);
	var vars = query.split('&');
	if (vars.length < 1 || vars[0].length < 1) return params;
	for (var i = 0; i < vars.length; i++) {
		var pair = vars[i].split('=');
		params[decodeURIComponent(pair[0])] = decodeURIComponent(pair[1]);
	}
	return params;
};

/**
 * Remove leading and trailing slashes from a string
 * @param  {String} str The string
 * @return {String}     The string with slashes removed
 */
var removeSlashes = function (str) {
	return '/' + str.replace(/^\/|\/$/g, '');
};

/**
 * Extract parameters from URL
 * @param  {String} match The URL to extract parameters from
 * @param  {Array}  names The parameter names
 * @return {Object}       The parameters as key/value pairs
 */
var regExpResultToParams = function (match, names) {
	if (!match || names.length === 0) return {};
	return match.slice(1, match.length).reduce(function (params, value, index) {
		params[names[index]] = decodeURIComponent(value);
		return params;
	}, {});
};

/**
 * Replace dynamic parts of a URL
 * @param  {String} route The URL path
 * @return {Object}       The regex and parameter names
 */
var replaceDynamicURLParts = function (route) {

	// Setup parameter names array
	var paramNames = [];

	// Parse route
	var regexp = new RegExp(route.replace(regPatterns.parameterRegex, function (full, dots, name) {
		paramNames.push(name);
		return regPatterns.replaceVarRegex;
	}).replace(regPatterns.wildcardRegex, regPatterns.replaceWildcard) + regPatterns.followedBySlashRegex, regPatterns.matchRegexFlags);

	return {
		regexp: regexp,
		paramNames: paramNames
	};

};

/**
 * If the URL points to '/', check for home route
 * @param  {Array} routes The routes
 * @return {Array}        The homepage route
 */
var getHome = function (routes) {
	var home = routes.filter(function (route) {
		return route.url === '/';
	});
	if (!home.length) return;
	return [{
		route: home[0],
		params: {}
	}];
};

/**
 * Find routes that match the pattern
 * @param  {String} url   The URL path
 * @param  {Array} routes The routes
 * @return {Array}        The matching routes
 */
var findMatchedRoutes = function (url, routes) {
	if (!routes || Reef._.trueTypeOf(routes) !== 'array') return [];
	if (url === '/') {
		var home = getHome(routes);
		if (home) return home;
	}
	return routes.map(function (route) {
		var parts = replaceDynamicURLParts(removeSlashes(route.url));
		var match = url.replace(/^\/+/, '/').match(parts.regexp);
		if (!match) return;
		var params = regExpResultToParams(match, parts.paramNames);
		return {
			route: route,
			params: params
		};
	}).filter(function (match) {
		return match;
	});
};

/**
 * Scroll to an anchor link
 * @param  {String} hash The anchor to scroll to
 * @param  {String} url  The URL to update [optional]
 */
var scrollToAnchor = function (hash, url) {
	if (url) {
		window.location.hash = '!/' + removeSlashes(url) + hash;
	}
	var elem = document.getElementById(hash.slice(1));
	if (!elem) return;
	elem.scrollIntoView();
	if (document.activeElement === elem) return;
	elem.setAttribute('tabindex', '-1');
	elem.focus();
};

/**
 * Get the href from a full URL, excluding hashes and query strings
 * @param  {URL}     url  The URL object
 * @param  {String}  root The root domain
 * @param  {Boolean} hash If true, as hash is used
 * @return {String}      The href
 */
var getHref = function (url, root, hash) {
	if ((hash && url.pathname.slice(-5) === '.html') || url.hash.indexOf('#!') === 0) {
		url = getLinkElem(url.hash.slice(2), root);
	}
	var href = removeSlashes(url.pathname);
	if (!root.length) return href;
	root = removeSlashes(root);
	if (href.indexOf(root) === 0) {
		href = href.slice(root.length);
	}
	return href;
};

/**
 * Get the route from the URL
 * @param  {URL} url      The URL object
 * @param  {Array} routes The routes
 * @param  {String} root  The domain root
 * @return {Object}       The matching route
 */
var getRoute = function (url, routes, root, hash) {
	var href = getHref(url, root, hash);
	var matches = findMatchedRoutes(href, routes);
	if (!matches.length) return;
	// if (matches[0].route.redirect) return getRoute(getLinkElem(matches[0].route.redirect, root, hash), routes, root, hash);
	var route = Reef.clone(matches[0].route);
	route.params = matches[0].params;
	route.search = getParams(url);
	return route;
};

/**
 * Emit an event before routing
 * @param  {Object} current The current route
 * @param  {Object} next    The next route
 */
var preEvent = function (current, next) {
	Reef.emit(window, 'beforeRouteUpdated', {
		current: current,
		next: next
	});
};

/**
 * Emit an event after routing
 * @param  {Object} current  The current route
 * @param  {Object} previous The previous route
 */
var postEvent = function (current, previous) {
	Reef.emit(window, 'routeUpdated', {
		current: current,
		previous: previous
	});
};

/**
 * Render any components using this router
 * @param  {Array} components Attached components
 */
var renderComponents = function (components) {
	components.forEach(function (component) {
		if (!('render' in component)) return;
		component.render();
	});
};

/**
 * Update the document title
 * @param  {Object}      route  The route object
 * @param  {Constructor} router The router component
 */
var updateTitle = function (route, router) {
	if (!route || !route.title) return;
	var title = typeof router.title === 'function' ? router.title(route) : router.title;
	document.title = title.replace('{{title}}', route.title);
};

/**
 * Bring heading into focus for AT announcements
 */
var focusHeading = function () {
	var heading = document.querySelector('h1, h2, h3, h4, h5, h6');
	if (!heading) return;
	if (!heading.hasAttribute('tabindex')) { heading.setAttribute('tabindex', '-1'); }
	heading.focus();
};

/**
 * Render an updated UI
 * @param  {URL}         link   The URL object
 * @param  {Constructor} router The router component
 * @param  {String}      hash   An anchor link to scroll to [optional]
 */
var render = function (route, router, hash) {

	// Render each component
	renderComponents(router._components);

	// a11y adjustments
	updateTitle(route, router);
	if (hash) {
		scrollToAnchor(hash);
	} else {
		focusHeading();
	}

};

/**
 * Update the route
 * @param  {URL}         link   The URL object
 * @param  {Constructor} router The router component
 */
var updateRoute = function (link, router) {

	// Get the route
	var route = getRoute(link, router.routes, router.root, router.hash);

	// If redirect, switch up
	if (route.redirect) return updateRoute(getLinkElem(typeof route.redirect === 'function' ? route.redirect(route) : route.redirect, router.root), router);

	// If hash enabled, handle anchors on URLs
	if (router.hash) {
		router.hashing = true;
		if (route.url === router.current.url && link.hash.length) {
			scrollToAnchor(link.hash, route.url);
			return;
		}
	}

	// Emit pre-routing event
	var previous = router.current;
	preEvent(previous, route);

	// Update the route
	router.current = route;

	// Get the href
	var href = removeSlashes(link.getAttribute('href'));

	// Update the URL
	if (router.hash) {
		window.location.hash = '!' + href;
	} else {
		history.pushState(route ? route : {}, route && route.title ? route.title : '', href);
	}

	// Render the UI
	render(route, router, link.hash);

	// Emit post-routing event
	postEvent(route, previous);

};

/**
 * Get the link from the event
 * @param  {Event} event The event object
 * @return {Node}        The link
 */
var getLink = function (event) {

	// Cache for better minification
	var elem = event.target;

	// If closest method is available, use it
	if ('closest' in elem) {
		return elem.closest('a');
	}

	// Otherwise, fallback to loop
	var eventPath = event.path || (event.composedPath ? event.composedPath() : null);
	if (!eventPath) return;
	for (var i = 0; i < eventPath.length; i++) {
		if (!eventPath[i].nodeName || eventPath[i].nodeName.toLowerCase() !== 'a' || !eventPath[i].href) continue;
		return eventPath[i];
	}

};

/**
 * Create a link element from a URL
 * @param  {String} url  The URL
 * @param  {String} root The root for the domain
 * @return {Node}       The element
 */
var getLinkElem = function (url, root) {
	var link = document.createElement('a');
	link.href = (root.length ? removeSlashes(root) : '') + removeSlashes(url);
	return link;
};

/**
 * Check if URL matches current location
 * @param  {String}  url The URL path
 * @return {Boolean}     If true, points to current location
 */
var isSamePath = function (url) {
	return url.pathname === window.location.pathname && url.search === window.location.search;
};

/**
 * Handle click events
 * @param  {Event}       event  The event object
 * @param  {Constructor} router The router component
 */
var clickHandler = function (event, router) {

	// Ignore for right-click or control/command click
	// Ignore if event was prevented
	if (event.metaKey || event.ctrlKey || event.shiftKey || event.defaultPrevented) return;

	// Check if a link was clicked
	// Ignore if link points to external location
	// Ignore if link has "download", rel="external", or "mailto:"
	var link = getLink(event);
	if (!link || link.host !== window.location.host || link.hasAttribute('download') || link.getAttribute('rel') === 'external' || link.href.indexOf('mailto:') > -1) return;

	// Make sure link isn't hash pointing to current URL
	if (isSamePath(link) && !router.hash && link.hash.length) return;

	// Stop link from running
	event.preventDefault();

	// Update the route
	updateRoute(link, router);

};

/**
 * Handle popstate events
 * @param  {Event}       event  The event object
 * @param  {Constructor} router The router component
 */
var popHandler = function (event, router) {
	if (!event.state) {
		history.replaceState(router.current, document.title, window.location.href);
	} else {

		// Emit pre-routing event
		var previous = router.current;
		preEvent(previous, event.state);

		// Update the UI
		router.current = event.state;
		render(router.current, router);

		// Emit post-routing event
		postEvent(event.state, previous);

	}
};

/**
 * Handle hashchange events
 * @param  {Event}       event  The event object
 * @param  {Constructor} router The router component
 */
var hashHandler = function (event, router) {

	// Don't run on hashchange transitions
	if (router.hashing) {
		router.hashing = false;
		return;
	}

	// Parse a link from the URL
	var link = getLinkElem(window.location.hash.slice(2), router.root);
	var href = link.getAttribute('href');
	var route = getRoute(link, router.routes, router.root, router.hash);

	// Emit pre-routing event
	var previous = router.current;
	preEvent(previous, route);

	// Update the UI
	router.current = route;
	render(route, router);

	// Emit post-routing event
	postEvent(route, previous);

};


//
// Constructor
//

/**
 * Router constructor
 * @param {Object} options The data store options
 */
Reef.Router = function (options) {

	// Make sure routes are provided
	if (!options || !options.routes || Reef._.trueTypeOf(options.routes) !== 'array' || !options.routes.length) return Reef._.err('Please provide an array of routes.');

	// Properties
	var _this = this;
	var _routes = options.routes;
	var _root = options.root ? options.root : '';
	var _title = options.title ? options.title : '{{title}}';
	var _components = [];
	var _hash = options.useHash || !support || window.location.protocol === 'file:';
	var _current = getRoute(window.location, _routes, _root, _hash);
	_this._hashing = false;

	// Event Handlers
	var _clickHandler = function (event) { clickHandler(event, _this); };
	var _popHandler = function (event) { popHandler(event, _this); };
	var _hashHandler = function (event) { hashHandler(event, _this); };

	// Create immutable property getters
	Object.defineProperties(_this, {
		routes: {value: Reef.clone(_routes, true)},
		root: {value: _root},
		title: {value: _title},
		hash: {value: _hash}
	});

	// Define setter and getter for current
	Object.defineProperty(_this, 'current', {
		get: function () {
			return Reef.clone(_current, true);
		},
		set: function (route) {
			_current = route;
			return true;
		}
	});

	// Define setter for _routes
	Object.defineProperty(_this, '_routes', {
		set: function (routes) {
			_routes = routes;
			return true;
		}
	});

	// Define setter for _routes
	Object.defineProperty(_this, '_components', {
		get: function () {
			return _components;
		}
	});

	// Initial setup
	updateTitle(_current, _this);

	// Listen for clicks and popstate events
	document.addEventListener('click', _clickHandler);
	if (_hash) {
		window.addEventListener('hashchange', _hashHandler);
	} else {
		history.replaceState(_current, document.title, window.location.href);
		window.addEventListener('popstate', _popHandler);
	}

};

/**
 * Add routes to the router
 * @param {Array|Object} routes The route or routes to add
 */
Reef.Router.prototype.addRoutes = function (routes) {
	var type = Reef._.trueTypeOf(routes);
	if (['array', 'object'].indexOf(type) < 0) return Reef._.err('Please provide a valid route or routes.');
	var _routes = this.routes;
	if (type === 'object') {
		_routes.push(routes);
	} else {
		_routes.concat(routes);
	}
	this._routes = _routes;
};

/**
 * Add a component to the router
 * @param {Reef} component A Reef component
 */
Reef.Router.prototype.addComponent = function (component) {
	this._components.push(component);
};

/**
 * Navigate to a path
 * @param  {String} url The URL to navigate to
 */
Reef.Router.prototype.navigate = function (url) {
	updateRoute(getLinkElem(url, this.root), this);
};
