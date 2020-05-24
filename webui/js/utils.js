function inject(str, obj) {
	return str.replace(/\${(.*?)}/g, function(match, submatch1) {
		return obj[submatch1];
	});
}

Function.prototype.extractComment = function() {
    var open = "/*";
    var close = "*/";
    var str = this.toString();

    var start = str.indexOf(open);
    var end = str.lastIndexOf(close);

    return str.slice(start + open.length, (str.length - end) * -1);
};

String.prototype.propsInjector = function(mod) {
	var tmpl = this;
	return function(props, router) {
		var mf = mod || function(props, router){return props}
		var modded = mf(Reef.clone(props), router);
		return inject(tmpl, modded);
	}
}

function jobType(num) {
  switch(num) {
    case 0:
      return 'Local'
    case 1:
      return 'Default'
    default:
      return 'Unknown'
  }
}
