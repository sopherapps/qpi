describe("Routing — Hash-Based Navigation", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
  });

  context("as a regular user", () => {
    beforeEach(() => {
      cy.visit("/");
      cy.get('input[type="text"]').clear().type("user@example.com");
      cy.get('input[type="password"]').clear().type("userpassword1234");
      cy.get('button[type="submit"]').click();
      cy.contains("h1", "QPI Interface").should("be.visible");
    });

    it("lands on the Overview tab by default", () => {
      cy.contains("h1", "Overview").should("be.visible");
      cy.contains("button", "Overview").should("have.class", "border-white");
    });

    it("navigates to Jobs Console via hash", () => {
      cy.visit("/#jobs");
      cy.contains("h1", "Jobs Console").should("be.visible");
      cy.contains("button", "Jobs Console").should("have.class", "border-white");
    });

    it("navigates to Bookings via hash", () => {
      cy.visit("/#bookings");
      cy.contains("h1", "Bookings").should("be.visible");
      cy.contains("button", "Bookings").should("have.class", "border-white");
    });

    it("navigates to QPU Registry via hash", () => {
      cy.visit("/#qpus");
      cy.contains("h1", "QPU Registry").should("be.visible");
      cy.contains("button", "QPU Registry").should("have.class", "border-white");
    });

    it("navigates to Settings via hash", () => {
      cy.visit("/#settings");
      cy.contains("h1", "Profile Settings").should("be.visible");
      cy.contains("button", "Profile Settings").should("have.class", "border-white");
    });

    it("shows 'Access Denied' when navigating to admin via hash", () => {
      cy.visit("/#admin");
      cy.contains("Access Denied").should("be.visible");
      // Admin panel button should still not be in the sidebar
      cy.contains("button", "Admin Panel").should("not.exist");
    });

    it("falls back to Overview for an invalid hash", () => {
      cy.visit("/#nonexistent");
      cy.contains("h1", "Overview").should("be.visible");
      cy.contains("button", "Overview").should("have.class", "border-white");
    });

    it("syncs browser back/forward with tab state", () => {
      // Start at overview
      cy.contains("h1", "Overview").should("be.visible");

      // Click to jobs
      cy.contains("button", "Jobs Console").click();
      cy.contains("h1", "Jobs Console").should("be.visible");
      cy.url().should("include", "#jobs");

      // Click to bookings
      cy.contains("button", "Bookings").click();
      cy.contains("h1", "Bookings").should("be.visible");
      cy.url().should("include", "#bookings");

      // Browser back → jobs
      cy.go("back");
      cy.contains("h1", "Jobs Console").should("be.visible");
      cy.url().should("include", "#jobs");

      // Browser back → overview
      cy.go("back");
      cy.contains("h1", "Overview").should("be.visible");

      // Browser forward → jobs
      cy.go("forward");
      cy.contains("h1", "Jobs Console").should("be.visible");
      cy.url().should("include", "#jobs");
    });
  });

  context("as an administrator", () => {
    beforeEach(() => {
      cy.visit("/");
      cy.contains("button", "Administrator").click();
      cy.get('input[type="text"]').clear().type("admin@example.com");
      cy.get('input[type="password"]').clear().type("supersecretpassword1234");
      cy.get('button[type="submit"]').click();
      cy.contains("h1", "QPI Interface").should("be.visible");
    });

    it("navigates to Admin Panel via hash", () => {
      cy.visit("/#admin");
      cy.contains("h1", "Admin Panel").should("be.visible");
      cy.contains("button", "Admin Panel").should("have.class", "border-white");
    });
  });
});
