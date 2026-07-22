describe("Driver Registry — Regular User View", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    // Log in as regular user
    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
  });

  it("does NOT show the Drivers nav item to non-admins", () => {
    cy.contains("button", "Drivers").should("not.exist");
  });

  it("keeps the Drivers page unreachable via direct hash navigation", () => {
    cy.visit("/#drivers");
    cy.contains("Access Denied.").should("be.visible");
  });
});
