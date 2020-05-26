function jobType(num) {
  switch (num) {
    case 0:
      return 'Local'
    case 1:
      return 'Default'
    default:
      return 'Unknown'
  }
}

function html(strings) {
  var args = arguments;
  return strings.reduce(function(acc, str, idx) {
    return acc + str + (args[idx + 1] || '');
  }, '')
}
