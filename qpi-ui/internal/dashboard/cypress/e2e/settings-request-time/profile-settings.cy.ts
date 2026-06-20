describe("Settings & Request Time — Profile Settings", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");
  });

  it("displays the regular user's email, quota, and role badge", () => {
    cy.get('input[type="text"]').type("user@example.com");
    cy.get('input[type="password"]').type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Profile Settings").click();
    cy.contains("h1", "Profile Settings").should("be.visible");

    // Email is shown
    cy.contains("user@example.com").should("be.visible");

    // Quota in seconds is displayed
    cy.contains("Allocated QPU Seconds")
      .parent()
      .within(() => {
        cy.get("span.font-mono").should("not.be.empty");
      });

    // Role badge shows "Standard User"
    cy.contains("Account Type")
      .parent()
      .within(() => {
        cy.contains("Standard User").should("be.visible");
      });
  });

  it("displays the admin's email, quota, and role badge", () => {
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').type("admin@example.com");
    cy.get('input[type="password"]').type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Profile Settings").click();
    cy.contains("h1", "Profile Settings").should("be.visible");

    // Email is shown
    cy.contains("admin@example.com").should("be.visible");

    // Quota in seconds is displayed
    cy.contains("Allocated QPU Seconds")
      .parent()
      .within(() => {
        cy.get("span.font-mono").should("not.be.empty");
      });

    // Role badge shows "Administrator"
    cy.contains("Account Type")
      .parent()
      .within(() => {
        cy.contains("Administrator").should("be.visible");
      });
  });
});
