describe("Overview Header — Theme Toggle", () => {
  beforeEach(() => {
    // Clear storage and cookies so we always start fresh
    cy.clearCookies();
    cy.clearLocalStorage();
    
    // Log in as an admin for testing header
    cy.visit("/", {
      onBeforeLoad: (win) => {
        // We do not set any theme in localStorage, so it should default to dark
        // Mock prefers-color-scheme to light just to prove our code defaults to dark anyway
        cy.stub(win, 'matchMedia').withArgs('(prefers-color-scheme: dark)').returns({ matches: false, addEventListener: () => {}, removeEventListener: () => {} });
      }
    });
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();

    // Verify dashboard is loaded
    cy.contains("h1", "QPI Interface").should("be.visible");
  });

  it("defaults to dark mode (has dark class on HTML)", () => {
    // We expect dark mode to be default regardless of OS theme
    cy.get('[data-testid="theme-toggle"]').should("be.visible");
    cy.get("html").should("have.class", "dark");
  });

  it("toggles the 'dark' class on the HTML element and persists it", () => {
    // 1. Initial State: Should be dark mode
    cy.get("html").should("have.class", "dark");
    cy.window().its("localStorage").invoke("getItem", "theme").should("be.null");

    // 2. Click Toggle: Should switch to light mode
    cy.get('[data-testid="theme-toggle"]').click();
    cy.get("html").should("not.have.class", "dark");
    cy.window().its("localStorage").invoke("getItem", "theme").should("eq", "light");

    // 3. Reload Page: Should stay in light mode
    cy.reload();
    cy.get("html").should("not.have.class", "dark");

    // 4. Click Toggle Again: Should switch back to dark mode
    cy.get('[data-testid="theme-toggle"]').click();
    cy.get("html").should("have.class", "dark");
    cy.window().its("localStorage").invoke("getItem", "theme").should("eq", "dark");
  });
});
