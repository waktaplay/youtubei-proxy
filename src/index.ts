import figlet from "figlet";

const PORT = 8123;

console.log(figlet.textSync("Innertube    Proxy", { font: "Standard" }));
console.log("----------------------------------------------------------------");

function copyHeader(headerName: string, to: Headers, from: Headers) {
  const hdrVal = from.get(headerName);
  if (hdrVal) {
    to.set(headerName, hdrVal);
  }
}

////////////////////////////////////////////////////////////
// #region Functions

const handler = async (req: Request) => {
  const { method, headers, url } = req;

  // If options send do CORS preflight
  if (method === "OPTIONS") {
    return new Response("", {
      status: 200,
      headers: {
        "Access-Control-Allow-Origin": headers.get("origin") || "*",
        "Access-Control-Allow-Methods": "*",
        "Access-Control-Allow-Headers":
          "Origin, X-Requested-With, Content-Type, Accept, Authorization, x-goog-visitor-id, x-goog-api-key, x-origin, x-youtube-client-version, x-youtube-client-name, x-goog-api-format-version, x-user-agent, Accept-Language, Range, Referer",
        "Access-Control-Max-Age": "86400",
        "Access-Control-Allow-Credentials": "true",
      },
    });
  }

  const urlObj = new URL(url, "http://localhost/");
  if (!urlObj.searchParams.has("__host")) {
    return new Response(
      "Request is formatted incorrectly. Please include __host in the query string.",
      { status: 400 }
    );
  }

  // Set the URL host to the __host parameter
  urlObj.host = urlObj.searchParams.get("__host")!;
  urlObj.protocol = "https";
  urlObj.port = "443";
  urlObj.searchParams.delete("__host");

  // Copy headers from the request to the new request
  const requestHeaders = new Headers(
    JSON.parse(urlObj.searchParams.get("__headers") || "{}")
  );
  copyHeader("range", requestHeaders, headers);

  if (!requestHeaders.has("user-agent")) {
    copyHeader("user-agent", requestHeaders, headers);
  }

  urlObj.searchParams.delete("__headers");

  // Make the request to the target server
  const fetchRes = await fetch(urlObj.toString(), {
    method,
    body: await req.text(),
    headers: requestHeaders,
    proxy:
      process.env.HTTP_PROXY ||
      process.env.HTTPS_PROXY ||
      process.env.PROXY ||
      undefined,
  });

  // Construct the return headers
  const responseHeaders = new Headers();

  // Copy content headers
  copyHeader("content-length", responseHeaders, fetchRes.headers);
  copyHeader("content-type", responseHeaders, fetchRes.headers);
  copyHeader("content-disposition", responseHeaders, fetchRes.headers);
  copyHeader("accept-ranges", responseHeaders, fetchRes.headers);
  copyHeader("content-range", responseHeaders, fetchRes.headers);

  // Add CORS headers
  responseHeaders.set(
    "Access-Control-Allow-Origin",
    headers.get("origin") || "*"
  );
  responseHeaders.set("Access-Control-Allow-Headers", "*");
  responseHeaders.set("Access-Control-Allow-Methods", "*");
  responseHeaders.set("Access-Control-Allow-Credentials", "true");

  // Return the proxied response
  return new Response(fetchRes.body, {
    status: fetchRes.status,
    headers: responseHeaders,
  });
};

// #endregion
////////////////////////////////////////////////////////////
// #region Initialise Server

Bun.serve({
  port: PORT,
  fetch: handler,
});

console.info(`[INFO] Server is running on port ${PORT}.`);

// #endregion
////////////////////////////////////////////////////////////
