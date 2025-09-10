// Example bgAttest implementation stub
// This is NOT a real Botguard solution. Replace with your own logic.

function bgAttest(input) {
  // input: { userAgent, pageURL, clientName, clientVersion, visitorID }
  // Produce some deterministic token placeholder for testing
  var base = [input.userAgent || '', input.clientName || '', input.clientVersion || '', input.visitorID || ''].join('|');
  var token = 'bg:' + String(base).substring(0, 32);
  return { token: token, ttlSeconds: 900 };
}



