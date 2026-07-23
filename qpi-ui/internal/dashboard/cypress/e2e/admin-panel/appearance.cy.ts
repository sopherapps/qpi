describe("Admin Panel — Appearance", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    // Log in as administrator
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();

    cy.contains("h1", "QPI Interface").should("be.visible");
    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");
    
    // Navigate to Appearance tab
    cy.contains("button", "Appearance").click();
  });

  it("loads the appearance tab and displays themes", () => {
    cy.contains("h3", "Dashboard Themes").should("be.visible");
    // Initially, there might be no themes
    cy.get("table").should("exist");
  });

  it("can open the Theme Editor for a new theme", () => {
    cy.contains("button", "Create New Theme").click();
    
    // Modal should appear
    cy.contains("h2", "Create New Theme").should("be.visible");
    
    // Check for inputs
    cy.contains("label", "Theme Name").should("exist");
    cy.contains("label", "Site Name").should("exist");
    
    // Close modal
    cy.contains("button", "✕").click();
    cy.contains("h2", "Create New Theme").should("not.exist");
  });
});
