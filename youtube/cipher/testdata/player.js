// A test JavaScript file that defines a decipher function.
function decipher(a) {
    a = a.split("");
    a = reverse(a);
    a = splice(a, 26);
    a = reverse(a);
    return a.join("");
}

// naive n-throttling decoder for tests (reverse string)
function ncode(n) {
    return n.split("").reverse().join("");
}

function reverse(a) {
    a.reverse();
    return a;
}

function splice(a, b) {
    a.splice(0, b);
    return a;
}
