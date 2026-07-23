describe("Theme Context — Custom CSS/JS Injection", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
  });

  it("injects custom CSS and executes custom JS when endpoints return content", () => {
    cy.intercept("GET", "/api/theme/active", {
      statusCode: 200,
      body: {
        id: "theme-custom-123",
        name: "Custom Theme",
        is_active: true,
        site_name: "QPI Custom",
        tagline: "Custom Injection Test",
        tokens: null,
      },
    }).as("getActiveTheme");

    cy.intercept("GET", "/api/theme/css", {
      statusCode: 200,
      headers: { "content-type": "text/css" },
      body: ".custom-css-test-class { color: rgb(255, 0, 0); }",
    }).as("getThemeCSS");

    cy.intercept("GET", "/api/theme/js", {
      statusCode: 200,
      headers: { "content-type": "text/javascript" },
      body: "window.__qpi_theme_js_injected = true;",
    }).as("getThemeJS");

    cy.visit("/");
    cy.wait("@getActiveTheme");
    cy.wait("@getThemeCSS");
    cy.wait("@getThemeJS");

    // Check style tag injection in head
    cy.get("head style#qpi-theme-css")
      .should("exist")
      .and("contain", ".custom-css-test-class { color: rgb(255, 0, 0); }");

    // Check script tag injection in body and script execution
    cy.get("body script#qpi-theme-js").should("exist");
    cy.window().its("__qpi_theme_js_injected").should("be.true");
  });

  it("handles 204 No Content for CSS and JS gracefully", () => {
    cy.intercept("GET", "/api/theme/active", {
      statusCode: 200,
      body: {
        id: "theme-default-456",
        name: "Default Theme",
        is_active: true,
        tokens: null,
      },
    }).as("getActiveTheme");

    cy.intercept("GET", "/api/theme/css", {
      statusCode: 204,
    }).as("getThemeCSS204");

    cy.intercept("GET", "/api/theme/js", {
      statusCode: 204,
    }).as("getThemeJS204");

    cy.visit("/");
    cy.wait("@getActiveTheme");
    cy.wait("@getThemeCSS204");
    cy.wait("@getThemeJS204");

    // Script element should not exist on 204
    cy.get("script#qpi-theme-js").should("not.exist");

    // Style element should either not exist or have empty text content
    cy.get("head").then(($head) => {
      const style = $head.find("style#qpi-theme-css");
      if (style.length > 0) {
        expect(style.text()).to.equal("");
      }
    });
  });

  it("does not re-fetch custom CSS/JS when toggling dark/light mode", () => {
    cy.intercept("GET", "/api/theme/active", {
      statusCode: 200,
      body: {
        id: "theme-custom-789",
        name: "Theme Toggle Test",
        is_active: true,
        tokens: null,
      },
    }).as("getActiveTheme");

    cy.intercept("GET", "/api/theme/css", {
      statusCode: 200,
      headers: { "content-type": "text/css" },
      body: ".toggle-test { margin: 0; }",
    }).as("getThemeCSS");

    cy.intercept("GET", "/api/theme/js", {
      statusCode: 200,
      headers: { "content-type": "text/javascript" },
      body: "window.__qpi_toggle_count = (window.__qpi_toggle_count || 0) + 1;",
    }).as("getThemeJS");

    cy.visit("/");
    cy.wait("@getThemeCSS");
    cy.wait("@getThemeJS");

    // Log in to access header theme toggle button
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();

    cy.get('[data-testid="theme-toggle"]').should("be.visible");

    // Toggle mode
    cy.get('[data-testid="theme-toggle"]').click();

    // Verify JS script was executed only once (not re-fetched on mode toggle)
    cy.window().its("__qpi_toggle_count").should("eq", 1);
    cy.get("@getThemeCSS.all").should("have.length", 1);
    cy.get("@getThemeJS.all").should("have.length", 1);
  });
});
