import "linkify-plugin-hashtag";
import "linkify-plugin-mention";
import linkifyHtml from "linkify-html";

export function linkifyContent(text: string): string {
  if (!text) return "";

  const options = {
    // Attributes to add to the generated <a> tags
    attributes: {
      rel: "noopener noreferrer",
    },
    // How to format the link text
    format: (value: string, type: string) => {
      if (type === "url" && value.length > 50) {
        return value.slice(0, 47) + "…";
      }
      return value;
    },
    // How to format the href for different link types
    formatHref: (href: string, type: string) => {
      if (type === "hashtag") {
        return `/explore?q=%23${encodeURIComponent(href.substring(1))}`; // Encode hashtag for URL safety
      }
      if (type === "mention") {
        return `/profile/${encodeURIComponent(href.substring(1))}`; // Encode username for URL safety
      }
      return href;
    },
    // Custom class names for different link types
    className: (href: string, type: string) => {
      if (type === "hashtag") return "text-link hashtag-link";
      if (type === "mention") return "text-link mention-link";
      return "text-link external-link";
    },
    target: (href: string, type: string) => {
      if (
        type === "url" &&
        !href.startsWith("/") &&
        !href.startsWith(window.location.origin)
      ) {
        return "_blank";
      }
      return "";
    },
    validate: true,
    nl2br: true,
  };

  return linkifyHtml(text, options);
}
