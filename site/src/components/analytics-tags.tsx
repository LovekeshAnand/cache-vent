// Public IDs, set in .env.production. With neither set, nothing loads.
const GA_ID = process.env.NEXT_PUBLIC_GA_ID ?? "";
const CLARITY_ID = process.env.NEXT_PUBLIC_CLARITY_ID ?? "";

/** AnalyticsTags are the standard GA4 and Clarity snippets, rendered into <head> so tag checkers see them. */
export function AnalyticsTags() {
  return (
    <>
      {GA_ID && (
        <>
          <script async src={`https://www.googletagmanager.com/gtag/js?id=${GA_ID}`} />
          <script
            dangerouslySetInnerHTML={{
              __html: `window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments)}gtag('js',new Date());gtag('config','${GA_ID}');`,
            }}
          />
        </>
      )}
      {CLARITY_ID && (
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(c,l,a,r,i,t,y){c[a]=c[a]||function(){(c[a].q=c[a].q||[]).push(arguments)};t=l.createElement(r);t.async=1;t.src="https://www.clarity.ms/tag/"+i;y=l.getElementsByTagName(r)[0];y.parentNode.insertBefore(t,y)})(window,document,"clarity","script","${CLARITY_ID}");`,
          }}
        />
      )}
    </>
  );
}
