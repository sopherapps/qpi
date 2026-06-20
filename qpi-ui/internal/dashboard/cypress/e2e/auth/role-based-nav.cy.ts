describe("Auth — Role-Based Navigation", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");
  });

  context("as a regular user", () => {
    beforeEach(() => {
      cy.get('input[type="text"]').clear().type("user@example.com");
      cy.get('input[type="password"]').clear().type("userpassword1234");
      cy.get('button[type="submit"]').click();
      cy.contains("h1", "QPI Interface").should("be.visible");
    });

    it("shows the standard navigation tabs", () => {
      cy.contains("button", "Overview").should("be.visible");
      cy.contains("button", "QPU Registry").should("be.visible");
      cy.contains("button", "Jobs Console").should("be.visible");
      cy.contains("button", "Bookings").should("be.visible");
      cy.contains("button", "Profile Settings").should("be.visible");
    });

    it("does NOT show the Admin Panel tab", () => {
      cy.contains("button", "Admin Panel").should("not.exist");
    });

    it("shows the QPU balance card with 'User Account' label", () => {
      cy.contains("span", "User Account").should("be.visible");
      cy.contains("span", "QPU Balance").should("be.visible");
    });

    it("shows the 'Request Time' link in the sidebar", () => {
      cy.contains("button", "Request Time").should("be.visible");
    });
  });

  context("as an administrator", () => {
    beforeEach(() => {
      cy.contains("button", "Administrator").click();
      cy.get('input[type="text"]').clear().type("admin@example.com");
      cy.get('input[type="password"]').clear().type("supersecretpassword1234");
      cy.get('button[type="submit"]').click();
      cy.contains("h1", "QPI Interface").should("be.visible");
    });

    it("shows all navigation tabs including Admin Panel", () => {
      cy.contains("button", "Overview").should("be.visible");
      cy.contains("button", "QPU Registry").should("be.visible");
      cy.contains("button", "Jobs Console").should("be.visible");
      cy.contains("button", "Bookings").should("be.visible");
      cy.contains("button", "Admin Panel").should("be.visible");
      cy.contains("button", "Profile Settings").should("be.visible");
    });

    it("shows the QPU balance card with 'Administrator' label", () => {
      cy.contains("span", "Administrator").should("be.visible");
    });

    it("does NOT show the 'Request Time' link", () => {
      cy.contains("button", "Request Time").should("not.exist");
    });
  });
});
